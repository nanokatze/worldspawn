package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"flag"
	"io"
	"log"
	"log/slog"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/go-json-experiment/json"

	"github.com/quic-go/quic-go"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/deathmatch/internal/replication"
	"worldspawn/internal/ecs"
	"worldspawn/internal/framing"
	"worldspawn/internal/gmath"
	"worldspawn/internal/nice"
)

var dataDir = flag.String("data", "cooked", "a")

// TODO: rewrite all of this garbage so that it's cleaner

// TODO: make all of the fields here private, probably...
type Server struct {
	mu sync.Mutex

	tickPeriod time.Duration

	scene *game.Scene

	// TODO: history should be a responsibility of game.Scene itself. Or maybe
	// not. Perhaps we should pass the history and shadow columns through
	// UpdateParams, but be responsible for managing those.
	//
	// TODO: we only need a subset of the entire world: only networked
	// components, no physics, etc.
	//
	// For lag compensation, we might need physics for queries, but it's not yet
	// clear how that should be done exactly.
	prevWorld *game.Scene

	mtimes modTimes

	users sync.Map // TODO: demo recorder and relay clients would live here as well.
}

// TODO: could we make it ignorant of game.Scene and game.Columns?
// TODO: rename
type modTimes struct {
	Entities []game.Time // TODO: rename to "Existence" or something
	Columns  [][]game.Time
}

// TODO: should we swap rows and columns?
func (mtimes *modTimes) Init(rows, columns int) {
	mtimes.Entities = make([]game.Time, rows)
	mtimes.Columns = make([][]game.Time, columns)
	for i := range mtimes.Columns {
		mtimes.Columns[i] = make([]game.Time, rows)
	}
}

// TODO: rename to conn?
// TODO: allow a single conn to control several entities at once
type user struct {
	// Perhaps we should have a component that does the mapping instead? (So it
	// would be kinda inverse...)
	//
	// TODO: actually use this field
	player ecs.ID

	// time is the latest timestamp (in game time) that we know the user
	// has the state for
	//
	// protected by WorldMu (TODO: ideally it shouldn't be that way)
	time game.Time

	send             chan struct{} // TODO: rename to make it clear that this is a sending backpressure semaphore
	marshaledUpdates chan []byte

	logger *slog.Logger
}

// TODO: we might wanna use early listener if that provides any benefits
func (s *Server) Serve(listener *quic.Listener, logger *slog.Logger) error {
	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			return err
		}

		go func() {
			// TODO: move this into a named func. We also need to put raddr into
			// a funny container as conn.RemoteAddr() can change over time
			// because of migration.
			logger := logger.With("laddr", conn.LocalAddr(), "raddr", conn.RemoteAddr())
			if err := s.serveConn(conn, logger); err != nil {
				logger.Warn("connection closed", "err", err)
			}
		}()
	}
}

func (s *Server) subscribe(u *user) {
	s.users.Store(u, struct{}{})
}

func (s *Server) unsubscribe(u *user) {
	s.users.Delete(u)
}

// We should split this up into two, one bit of code that runs on auth on a
// quic.Stream and one that runs after on raw quic.Connection, comparable to
// HTTP/3 requests being upgraded to webtransport sessions.
// TODO: return an error and log it at the upper level, also have the caller
// pass the logger (or the entire user) in here
func (s *Server) serveConn(conn *quic.Conn, logger *slog.Logger) error {
	// The code here is as-if we upgraded to webtransport session.
	u := &user{
		send:             make(chan struct{}, 2),
		marshaledUpdates: make(chan []byte, 1), // one less than cap(send)
		logger:           logger,
	}

	logger.Info("connected")

	defer func() {
		// TODO: should be done earlier
		conn.CloseWithError(0, "fuck off")
	}()

	stream, err := conn.OpenUniStream()
	if err != nil {
		return err
	}

	stream2 := stream

	// TODO: we should identify our streams somehow

	// TODO: we shouldn't need to spawn the player immediately. The game should
	// decide when to do so
	s.mu.Lock()
	u.player = game.SpawnPlayer(s.scene, &game.UpdateParams{Logger: slog.Default()})
	s.mu.Unlock()

	framer := framing.NewFramer(stream2)

	{
		binary.Write(framer, binary.LittleEndian, int64(replication.ResetTicker))
		binary.Write(framer, binary.LittleEndian, s.tickPeriod)
		framer.Next()
	}

	// TODO: remove this and let world delta updates take care of this? That
	// might introduce some delay so perhaps not.
	{
		binary.Write(framer, binary.LittleEndian, int64(replication.ResetScene))
		binary.Write(framer, binary.LittleEndian, int64(s.scene.Cap()))
		framer.Next()

		defer func() {
			s.mu.Lock()
			defer s.mu.Unlock()

			// TODO: don't kill Character here
			player, _ := game.SceneGetEntity[game.Player](s.scene, u.player)
			s.scene.Delete.Set(player.ControlledCharacter, struct{}{})
			s.scene.Delete.Set(u.player, struct{}{})
		}()
	}

	{
		binary.Write(framer, binary.LittleEndian, int64(replication.SetPlayer))
		binary.Write(framer, binary.LittleEndian, u.player)
		framer.Next()
	}

	// TODO: we could optionally eagerly send this user a snapshot right now,
	// for less join latency

	// stream2.Flush()

	done := make(chan struct{})
	defer close(done)

	go func() {
		for {
			framer := framing.NewFramer(stream2)

			// TODO: instrument with counting writers and everything so we can
			// see what's happening.

			// TODO: move this into a function

			func() {
				defer func() { <-u.send }()

				select {
				case buf := <-u.marshaledUpdates:
					binary.Write(framer, binary.LittleEndian, uint64(replication.UpdateScene))
					framer.Write(buf)
					framer.Next()
				case <-done:
					return
				}
			}()
		}
	}()

	s.subscribe(u)
	defer s.unsubscribe(u)

	// TODO: also call disconnection handler which should remove the player
	// entity

	// TODO: accept streams in a loop (perhaps spin up a goroutine to accept
	// each kind of stream) and spawn a goroutine for each one.
	inputStream, err := conn.AcceptUniStream(context.Background())
	if err != nil {
		return err
	}

	if err := s.handleInputPackets(u, inputStream); err != nil {
		return err
	}

	return nil
}

// TODO: rename to receiveInputCommands perhaps?
func (s *Server) handleInputPackets(u *user, stream io.Reader) error {
	deframer := framing.NewDeframer(stream)
	for {
		// TODO: we don't need double layering of messages and input packets. I
		// guess we could have a message per input packet?

		var cmds []game.TimestampedInputCmd
		if err := readInputCmds(deframer, &cmds); err != nil {
			return err
		}

		// TODO: let's apply the inputs at the beginning of the tick and just
		// collect them into some buffer(s) here. We could perhaps sort packets
		// and randomly shuffle ones that fall at the same time (or maybe we
		// shouldn't bother) for consistency and tie breaking purposes. Or maybe
		// just shuffle the players for which we'll be applying inputs.
		func() {
			// TODO: we're not super intelligent about it

			s.mu.Lock()
			defer s.mu.Unlock()

			// u.time = max(u.time, tmpTime)

			for _, cmd := range cmds {
				s.scene.HandleInput(u.player, cmd, &game.UpdateParams{Δt: s.tickPeriod, Logger: slog.Default()})
			}
		}()

		if err := deframer.Next(); err != nil {
			return err
		}
	}
}

func (mtimes *modTimes) update(prevWorld, scene *game.Scene) {
	{
		a := prevWorld.Table.IDs()
		b := scene.Table.IDs()

		for i := range b.Cap() {
			if a.Index(i) != b.Index(i) {
				mtimes.Entities[i] = scene.Now
			}
		}
	}

	for _, columnIndex := range replication.ReplicatedColumns {
		old := reflect.ValueOf(&prevWorld.Columns).Elem().Field(columnIndex).Addr().Interface().(ecs.AnyColumn).Reflect()
		cur := reflect.ValueOf(&scene.Columns).Elem().Field(columnIndex).Addr().Interface().(ecs.AnyColumn).Reflect()

		t := cur.ElemType()

		// TODO: once we get reactivity, only iterate over things that could've
		// changed.
		// TODO: stop using reflect.DeepEqual, it's not exactly fast
		oldc := reflect.New(t)
		curc := reflect.New(t)
		for id := range cur.All() {
			equal := false
			if old.Get(id, oldc.Elem()) {
				cur.Get(id, curc.Elem())
				equal = reflect.DeepEqual(oldc.Interface(), curc.Interface())
			}
			if !equal {
				mtimes.Columns[columnIndex][id.Index()] = scene.Now
			}
		}
	}
}

func (s *Server) tick(Δt time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.scene.Step(&game.UpdateParams{Δt: Δt, Logger: slog.Default()})

	s.mtimes.update(s.prevWorld, s.scene)

	// TODO: run SendUpdates in parallel
	s.users.Range(func(u, _ any) bool {
		// TODO: run s.sendUpdates in their own goroutines
		s.sendUpdates(u.(*user))
		return true
	})

	// Copy the current s.World to s.PrevWorld

	// TODO: move this into a method on the World
	s.prevWorld.Now = s.scene.Now
	s.prevWorld.Table.Copy(s.scene.Table)
	for _, columnIndex := range replication.ReplicatedColumns {
		dst := reflect.ValueOf(&s.prevWorld.Columns).Elem().Field(columnIndex).Addr().Interface().(ecs.AnyColumn).Reflect()
		src := reflect.ValueOf(&s.scene.Columns).Elem().Field(columnIndex).Addr().Interface().(ecs.AnyColumn).Reflect()
		dst.Copy(src)
	}
}

// Must be called with u.server.WorldMu held
func (s *Server) sendUpdates(u *user) {
	// This code is really alloc-happy right now, we'll work on minimizing that

	select {
	case u.send <- struct{}{}:

	default:
		return
	}

	// TODO: we could use our own Buffer with Seek depending on how we're going
	// to arrange things (where compression will happen etc)
	buf := new(bytes.Buffer)
	enc := nice.NewEncoder(buf, replication.NiceOptions)

	if err := nice.MarshalEncode(enc, &s.scene.Now); err != nil {
		panic(err)
	}

	// Send everything that changed since u.latestAcked

	{
		// TODO: we should seek instead of doing extra allocations I think...

		buf2 := new(bytes.Buffer)
		enc2 := nice.NewEncoder(buf2, replication.NiceOptions)

		n := 0
		for i, mtime := range s.mtimes.Entities {
			if mtime.Before(u.time) {
				continue
			}

			id := s.scene.Table.IDs().Index(i)

			gen := ^uint32(0)
			if id != 0 {
				gen = id.Generation()
			}

			indexGen := struct{ Index, Gen uint32 }{Index: uint32(i), Gen: gen}

			if err := nice.MarshalEncode(enc2, &indexGen); err != nil {
				panic(err)
			}

			n++
		}

		if err := nice.MarshalEncode(enc, &n); err != nil {
			panic(err)
		}
		io.Copy(buf, buf2)
	}

	// TODO: clean this horribleness up

	// TODO: insert various canaries to make debugging easier

	for _, columnIndex := range replication.ReplicatedColumns {
		column := reflect.ValueOf(&s.scene.Columns).Elem().Field(columnIndex).Addr().Interface().(ecs.AnyColumn).Reflect()

		buf2 := new(bytes.Buffer)
		enc2 := nice.NewEncoder(buf2, replication.NiceOptions)

		n := 0
		v := reflect.New(column.ElemType())
		for i, mtime := range s.mtimes.Columns[columnIndex] {
			if mtime.Before(u.time) {
				continue
			}

			id := s.scene.Table.IDs().Index(i)
			if id == 0 {
				continue
			}

			exists := column.Get(id, v.Elem())

			indexAndOk := uint32(i) // TODO: rename
			if exists {
				indexAndOk |= 1 << 31
			}

			if err := nice.MarshalEncode(enc2, &indexAndOk); err != nil {
				panic(err)
			}
			if exists {
				if err := nice.MarshalEncode(enc2, v.Interface()); err != nil {
					panic(err)
				}
			}

			n++
		}

		if err := nice.MarshalEncode(enc, &n); err != nil {
			panic(err)
		}
		io.Copy(buf, buf2)
	}

	// Because we're sending stuff reliably, we can do this.
	//
	// TODO: set this if we're actually sending stuff.
	u.time = s.scene.Now

	// TODO: we could compress now or later. Compressing now means we increase
	// the server's critical section, compressing later means our memory usage
	// is higher. Also, if we compress later, we could also run compressor
	// transparently on the underlying stream. This could improve compression
	// across ticks, but makes us worse off if we ever decide we want to
	// dynamically switch between serializing per-user and serialize once for
	// all users.
	u.marshaledUpdates <- buf.Bytes()
}

// TODO: move to common util package
type CountingWriter struct {
	W io.Writer
	N int64
}

func (w *CountingWriter) Write(b []byte) (int, error) {
	n, err := w.W.Write(b)
	w.N += int64(n)
	return n, err
}

// TODO: remove this function in favor of just having the caller use
// nice directly?
func readInputCmds(r io.Reader, cmds *[]game.TimestampedInputCmd) error {
	dec := nice.NewDecoder(r, replication.NiceOptions)
	return nice.UnmarshalDecode(dec, cmds)
}

func main() {
	flag.Parse()

	conf := &Config{}

	if f, err := os.Open("serverconfig.json"); err == nil {
		if err := json.UnmarshalRead(f, conf); err != nil {
			panic(err)
		}
	}

	// TODO: reformulate ourselves in terms of a http server? this would require
	// that there's redundancy (e.g. erasure correction, like
	// https://datatracker.ietf.org/doc/draft-michel-quic-fec/) extensions to
	// QUIC adopted first so that we can keep latency low

	// TODO: pull from config
	tlsCert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatal(err)
	}

	quicConfig := &quic.Config{
		MaxConnectionReceiveWindow: 1 << 17,
		MaxIncomingStreams:         -1,
		MaxIncomingUniStreams:      4,
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"worldspawn"},
	}

	// TODO: pull the listen address from config
	listener, err := quic.ListenAddr(":32017", tlsConfig, quicConfig)
	if err != nil {
		log.Fatal(err)
	}

	game.Data = os.DirFS(*dataDir)

	s := new(Server)

	maxEntities := 1000

	s.tickPeriod = time.Second / 64
	s.scene = game.NewScene(maxEntities)
	s.prevWorld = game.NewScene(maxEntities)
	s.mtimes.Init(maxEntities, reflect.TypeFor[game.Columns]().NumField())

	sceneFile, err := game.Data.Open(conf.MapRotation[0])
	if err := s.scene.Restore(sceneFile); err != nil {
		log.Fatalf("restore %v: %v", sceneFile, err)
	}
	sceneFile.Close()

	info := &game.UpdateParams{Logger: slog.Default()}

	if true {
		test := s.scene.CreateEntity(info)
		s.scene.SetTransform(test, gmath.TRS3f64{
			T: gmath.Vec3f64{0, -1, 0},
			R: gmath.Rot3One(),
			S: gmath.Mat3x3UOne[float32](),
		})
		s.scene.Skeleton.Set(test, "testcharacter4/skeletons/metarig")
		s.scene.RenderingGeometry.Set(test, "testcharacter4/geometries/TestCharacter4")
		s.scene.Entity.Set(test, game.Animtest{"testcharacter4/animations/metarigAction"})

		test2 := s.scene.CreateEntity(info)
		s.scene.SetParent(test2, test)
		s.scene.ParentBone.Set(test2, "hand.L")
		s.scene.SetTransform(test2, gmath.TRS3f64{
			T: gmath.Vec3f64{0, 0.4, 0.1},
			R: gmath.Rot3One(),
			S: gmath.Mat3x3UOne[float32](),
		})
		s.scene.RenderingGeometry.Set(test2, "weapons/grenade_launcher/geometries/Grenade_Launcher")
		s.scene.Entity.Set(test2, game.Testburger{BaseColor: [4]float32{0.8, 0.8, 0.8, 1}})
	}

	if false {
		{
			weapon := s.scene.CreateEntity(info)
			s.scene.Entity.Set(weapon, game.WeaponGrenadeLauncher{})

			// Aaand "drop" the weapon

			dropped := s.scene.CreateDroppedWeapon(weapon, info)
			s.scene.SetTransform(dropped, gmath.TRS3f64{
				T: gmath.Vec3f64{0, 0, 1},
				R: gmath.Rot3One(),
				S: gmath.Mat3x3UOne[float32](),
			})
		}

		{
			weapon := s.scene.CreateEntity(info)
			s.scene.Entity.Set(weapon, game.WeaponSniperRifle{})

			dropped := s.scene.CreateDroppedWeapon(weapon, info)
			s.scene.SetTransform(dropped, gmath.TRS3f64{
				T: gmath.Vec3f64{0, -10, 1},
				R: gmath.Rot3One(),
				S: gmath.Mat3x3UOne[float32](),
			})
		}
	}

	s.scene.InstantinateCollections()

	// Reset mtimes
	for _, comp := range s.mtimes.Columns {
		for j := range comp {
			comp[j] = s.scene.Now
		}
	}

	go func() {
		// TODO: make the ticker a member of Server
		ticker := time.NewTicker(s.tickPeriod)

		for {
			<-ticker.C

			s.tick(s.tickPeriod)
		}
	}()

	logger := slog.Default() // .With("uuid", 42)

	if err := s.Serve(listener, logger); err != nil {
		logger.With("listenaddr", listener.Addr()).Error("serve", "err", err)
	}
}
