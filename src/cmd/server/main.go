package main

// #cgo LDFLAGS: -lphysics
// #cgo LDFLAGS: -lJolt
// #cgo LDFLAGS: -lm -lstdc++
import "C"

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"io"
	"log"
	"log/slog"
	rand "math/rand/v2"
	"os"
	"reflect"
	"runtime/trace"
	"sync"
	"time"

	"github.com/go-json-experiment/json"

	"github.com/quic-go/quic-go"

	"worldspawn"
	"worldspawn/ecs"
	"worldspawn/experiments/encoding/nice"
	"worldspawn/geometry-go"
	"worldspawn/protocol"
)

var dataDir = flag.String("data", "data/cooked", "a")

// TODO: rewrite all of this garbage so that it's cleaner

var _ = hex.Dump

// TODO: make all of the fields here private, probably...
type Server struct {
	mu sync.Mutex

	Δt time.Duration

	maxEntities int

	world *worldspawn.World

	// TODO: we only need a subset of the entire world: only networked
	// components, no physics, etc.
	//
	// For lag compensation, we might need physics for queries, but it's not yet
	// clear how that should be done exactly.
	prevWorld *worldspawn.World

	dirty dirty

	users sync.Map // TODO: demo recorder and relay clients would live here as well.
}

// TODO: inline into the Server
type dirty struct {
	// TODO: should we have a linked list of IDs and place newest updated IDs at
	// the beginning of the list?

	IDAlloc []worldspawn.Time

	Components [][]worldspawn.Time
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
	time worldspawn.Time

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
	u.player = spawnplayer(s.world)
	s.mu.Unlock()

	framer := protocol.NewFramer(stream2)

	{
		binary.Write(framer, binary.LittleEndian, int64(protocol.SetDeltaTime))
		binary.Write(framer, binary.LittleEndian, s.Δt)
		framer.Next()
	}

	// TODO: remove this and let world delta updates take care of this? That
	// might introduce some delay so perhaps not.
	{
		binary.Write(framer, binary.LittleEndian, int64(protocol.ResetWorld))
		binary.Write(framer, binary.LittleEndian, int64(s.maxEntities))
		framer.Next()

		defer func() {
			s.mu.Lock()
			defer s.mu.Unlock()

			// TODO: depending on the type of game, removing the entity is not
			// necessarily what we want (e.g. we might want to let the AI take
			// over the control, or put the character to sleep, or ...)
			s.world.Delete.Store(u.player, struct{}{})
		}()
	}

	{
		binary.Write(framer, binary.LittleEndian, int64(protocol.SetPlayer))
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
			framer := protocol.NewFramer(stream2)

			// TODO: instrument with counting writers and everything so we can
			// see what's happening.

			// TODO: move this into a function

			func() {
				defer func() { <-u.send }()

				select {
				case buf := <-u.marshaledUpdates:
					binary.Write(framer, binary.LittleEndian, uint64(protocol.UpdateWorld))
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
	deframer := protocol.NewDeframer(stream)
	for {
		// TODO: we don't need double layering of messages and input packets. I
		// guess we could have a message per input packet?

		var cmds []worldspawn.TimestampedInputCmd
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
				s.world.HandleInput(u.player, cmd, &worldspawn.UpdateParams{Δt: s.Δt, Logger: slog.Default()})
			}
		}()

		if err := deframer.Next(); err != nil {
			return err
		}
	}
}

func (s *Server) tick(Δt time.Duration) {
	defer trace.StartRegion(context.Background(), "Tick").End()

	s.mu.Lock()
	defer s.mu.Unlock()

	trace.WithRegion(context.Background(), "World Update", func() {
		s.world.Update(&worldspawn.UpdateParams{Δt: s.Δt, Logger: slog.Default()})
	})

	// update s.dirty. TODO: move into its own code?

	{
		a := s.prevWorld.IDAlloc
		b := s.world.IDAlloc

		for i := range b.Cap() {
			if a.Index(i) != b.Index(i) {
				s.dirty.IDAlloc[i] = s.world.Now
			}
		}
	}

	for i, comp := range comps {
		old := ecs.Reflect(reflect.ValueOf(&s.prevWorld.Components).
			Elem().FieldByName(comp).Addr().Interface())
		cur := ecs.Reflect(reflect.ValueOf(&s.world.Components).
			Elem().FieldByName(comp).Addr().Interface())

		t := cur.ElemType()

		// TODO: once we get reactivity, only iterate over things that could've
		// changed.
		// TODO: stop using reflect.DeepEqual, it's not exactly fast
		oldc := reflect.New(t)
		curc := reflect.New(t)
		for id := range cur.All() {
			equal := false
			if old.Load(id, oldc.Elem()) {
				cur.Load(id, curc.Elem())
				equal = reflect.DeepEqual(oldc.Interface(), curc.Interface())
			}
			if !equal {
				s.dirty.Components[i][id.Index()] = s.world.Now
			}
		}
	}

	// TODO: run SendUpdates in parallel
	s.users.Range(func(u, _ any) bool {
		// TODO: run s.sendUpdates in their own goroutines
		s.sendUpdates(u.(*user))
		return true
	})

	// TODO: any reason not to clear transients *before* sending updates, etc? I
	// guess demo recording would want to record the transients. Should we
	// separate demo recording, relay (tv) and normal replication?
	worldspawn.ClearTransientComponents(s.world)

	// Copy the current s.World to s.PrevWorld

	// TODO: move this into a method on the World
	s.prevWorld.Now = s.world.Now
	s.prevWorld.IDAlloc.Copy(s.world.IDAlloc)
	for _, comp := range comps {
		dst := ecs.Reflect(reflect.ValueOf(&s.prevWorld.Components).Elem().FieldByName(comp).Addr().Interface())
		src := ecs.Reflect(reflect.ValueOf(&s.world.Components).Elem().FieldByName(comp).Addr().Interface())
		ecs.CopyComponentStore(dst, src)
	}

	s.world.Now = s.world.Now.Add(Δt)
}

// TODO: remove in favor of struct tags
//
// So imo what we should actually do is have a table of *AnyComponentStores or
// w/e, e.g. the World could implement some kind of interface which we could to
// use to get AnyComponentStore for each component store that needs to be
// de/serialized in that manner. This code should not assume anything about the
// struct layout.
var comps = []string{
	"Entity",
	"TranslationRotation",
	"Scale",
	"RenderingGeometry",
	"SoundEffect",
	"CosmeticOffset",
	"Animation", // stress test map nice un/marshaling
	"Pose",      // stress test map nice un/marshaling
	"CollisionGeometry",
	"CollisionLayer",
	"PhysicsFilter",
	"GravityFactor",
	"PhysicsMassOverride",
	"PhysicsInertiaOverride",
	"ArmedCharacter",
	"ViewPunch",
	"ViewPunchVelocity",
	"Viewmodel",
	"PlayerSpawn",
	"ResetCosmeticOffsetOnContact",
	"SpawnTime",
	"DeleteAfter",
}

// called with u.server.WorldMu held
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
	enc := nice.NewEncoder(buf, worldspawn.WorldNiceOptions)

	{
		// TODO: only send SingletonComponents that changed
		if err := nice.MarshalEncode(enc, &s.world.SingletonComponents); err != nil {
			panic(err)
		}
	}

	// Send everything that changed since u.latestAcked

	{
		// TODO: we should seek instead of doing extra allocations I think...

		buf2 := new(bytes.Buffer)
		enc2 := nice.NewEncoder(buf2, worldspawn.WorldNiceOptions)

		n := 0
		for i, dirtied := range s.dirty.IDAlloc {
			if dirtied.Before(u.time) {
				continue
			}

			id := s.world.IDAlloc.Index(i)

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

	for idx, comp := range comps {
		cs := ecs.Reflect(reflect.ValueOf(&s.world.Components).Elem().FieldByName(comp).Addr().Interface())

		buf2 := new(bytes.Buffer)
		enc2 := nice.NewEncoder(buf2, worldspawn.WorldNiceOptions)

		n := 0
		v := reflect.New(cs.ElemType())
		for i, dirtied := range s.dirty.Components[idx] {
			if dirtied.Before(u.time) {
				continue
			}

			id := s.world.IDAlloc.Index(i)
			if id == 0 {
				continue
			}

			exists := cs.Load(id, v.Elem())

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
	u.time = s.world.Now

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
func readInputCmds(r io.Reader, cmds *[]worldspawn.TimestampedInputCmd) error {
	dec := nice.NewDecoder(r, nice.WithUnmarshaler(worldspawn.InputCommandNiceUnmarshal))
	return nice.UnmarshalDecode(dec, cmds)
}

func spawnplayer(w *worldspawn.World) ecs.ID {
	player := w.IDAlloc.Alloc()

	var playerSpawns []ecs.ID
	for id := range w.PlayerSpawn.All() {
		playerSpawns = append(playerSpawns, id)
	}

	w.Entity.Store(player, worldspawn.FPSCharacter{
		WalkVelocity:                21.6 / 3.6,
		BackwardsWalkVelocityFactor: 0.8,
		WalkAcceleration:            35,
		JumpVelocity:                4,
		StandingViewHeight:          1.9 - 0.1,
	})

	// meh
	t, _ := w.TranslationRotation.Load(playerSpawns[rand.IntN(len(playerSpawns))])
	w.TranslationRotation.Store(player, t)

	w.Scale.Store(player, geometry.Vec3Broadcast(1))
	w.RenderingGeometry.Store(player, worldspawn.RenderingGeometry{Filename: "testcharacter4/geometries/TestCharacter4"})
	w.CollisionGeometry.Store(player, worldspawn.CollisionGeometry{
		Translation: geometry.Vec3{0, 0, 1.9 / 2},
		Rotation:    geometry.Rot3One(),
		Scale:       geometry.Vec3Broadcast(1),

		Kind:         worldspawn.PhysicsShapeCylinder,
		HalfExtent:   geometry.Vec3{1, 1, 0}.Scale(0.4).Add(geometry.Vec3{0, 0, 1.9 / 2}),
		ConvexRadius: 0.0,
	})
	w.CollisionLayer.Store(player, worldspawn.PhysicsLayerMovingKinematic)
	w.PhysicsMassOverride.Store(player, 100)

	slots := []ecs.ID{}

	{
		gun := w.SpawnPrefab(worldspawn.PrefabRef{Filename: "weapons/grenade_launcher/grenade_launcher.json"})
		w.TranslationRotation.Store(gun, worldspawn.TranslationRotationOne())
		w.Scale.Store(gun, geometry.Vec3Broadcast(1))

		slots = append(slots, gun)
	}

	{
		gun := w.SpawnPrefab(worldspawn.PrefabRef{Filename: "weapons/rocket_launcher/rocket_launcher.json"})
		w.TranslationRotation.Store(gun, worldspawn.TranslationRotationOne())
		w.Scale.Store(gun, geometry.Vec3Broadcast(1))

		slots = append(slots, gun)
	}

	if false {
		gun := w.SpawnPrefab(worldspawn.PrefabRef{Filename: "weapons/sniper_rifle/sniper_rifle.json"})
		w.TranslationRotation.Store(gun, worldspawn.TranslationRotationOne())
		w.Scale.Store(gun, geometry.Vec3Broadcast(1))

		slots = append(slots, gun)
	}

	w.ArmedCharacter.Store(player, worldspawn.ArmedCharacter{Slots: slots})

	return player
}

func main() {
	flag.Parse()

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

	listener, err := quic.ListenAddr(":32017", tlsConfig, quicConfig)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: set up a control endpoint (JSON REST API with commands to dump the
	// current state, etc)

	worldspawn.Data = os.DirFS(*dataDir)

	s := new(Server)

	s.Δt = time.Second / 64
	s.maxEntities = 1000
	s.world = worldspawn.NewWorld(s.maxEntities)
	s.prevWorld = worldspawn.NewWorld(s.maxEntities)
	s.dirty.IDAlloc = make([]worldspawn.Time, s.maxEntities)
	s.dirty.Components = make([][]worldspawn.Time, len(comps))
	for i := range len(comps) {
		s.dirty.Components[i] = make([]worldspawn.Time, s.maxEntities)
	}

	// TODO: move loading into worldspawn? It needs to handle instancing
	// collections, so that seems like it would be the right place for it.
	// Alternatively we can instance collections in an ad-hoc manner.
	sceneFile, err := worldspawn.Data.Open("maps/lockdown/scenes/Scene")
	if err != nil {
		log.Fatal("newSinglePlayerSession: ", err)
	}
	if err := json.UnmarshalRead(sceneFile, s.world, worldspawn.WorldJSONOptions); err != nil {
		log.Fatalf("newSinglePlayerSession %v: %v", sceneFile, err)
	}
	sceneFile.Close()

	s.world.InstantinateCollections()

	s.world.Now = max(s.world.Now, 1)

	// Reset dirty times
	for _, comp := range s.dirty.Components {
		for j := range comp {
			comp[j] = s.world.Now
		}
	}

	for _, comp := range s.dirty.Components {
		for _, dirtied := range comp {
			if dirtied == 0 || dirtied.After(s.world.Now) {
				panic("dirtied should never be 0 nor in the future")
			}
		}
	}

	go func() {
		ticker := time.NewTicker(s.Δt)
		defer ticker.Stop()

		for {
			<-ticker.C

			s.tick(s.Δt)
		}
	}()

	logger := slog.Default() // .With("uuid", 42)

	if err := s.Serve(listener, logger); err != nil {
		logger.With("listenaddr", listener.Addr()).Error("serve", "err", err)
	}
}
