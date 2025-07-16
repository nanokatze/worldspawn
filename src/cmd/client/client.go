package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"io"
	"log"
	"log/slog"
	"reflect"
	"runtime/trace"
	"sync"
	"time"

	"github.com/quic-go/quic-go"

	"worldspawn"
	"worldspawn/ecs"
	"worldspawn/experiments/encoding/nice"
	"worldspawn/protocol"
	// "worldspawn/geometry-go"
)

var _ = hex.Dump

// TODO: make both calls usable for updating other state (notably action
// sets/layers but also gamepad LEDs/rumble?) Actually LEDs/rumble would be figured out
// TODO: shrink the interface so that we only pass the components that the
// renderer cares about
// TODO: reformulate the interface in terms of renderer's scene instead of
// worldspawn component stores?
type Renderer interface {
	Tick(w *worldspawn.World, camera ecs.ID, t0, t1 worldspawn.Time, frameDuration time.Duration)
	Subtick(w *worldspawn.World, camera ecs.ID)
}

type Client struct {
	done chan struct{} // TODO: rename to something like "close"

	mu sync.Mutex

	conn        *quic.Conn
	inputStream *quic.SendStream

	inputCmds []worldspawn.TimestampedInputCmd

	Δt    time.Duration
	world *worldspawn.World

	// TODO: we could be in control of many player entities
	player ecs.ID

	renderer Renderer
}

// TODO: let Client take Connection instead of addr, and make it a bit more
// abstract, so that we can use the same construction for single player and demo
// playback sessions?
func newClient(renderer Renderer, addr string) (*Client, error) {
	// TODO: contextualize log messages
	slog.Info("dial", "addr", addr)

	s := new(Client)
	s.done = make(chan struct{})
	s.renderer = renderer

	quicConfig := &quic.Config{
		MaxIncomingStreams:    -1, // we only use unidirectional streams, though we probably should use a bidirectional stream for the game.
		MaxIncomingUniStreams: 4,
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"worldspawn"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := quic.DialAddr(ctx, addr, tlsConfig, quicConfig)
	if err != nil {
		log.Fatal(err)
	}
	s.conn = conn

	s.inputStream, _ = conn.OpenUniStream()

	stream, err := conn.AcceptUniStream(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// stream2, _ := zstd.NewReader(stream)
	stream2 := stream

	ready := make(chan struct{})
	go func() {
		deframer := protocol.NewDeframer(stream2)
		for {
			var msgtype uint64
			if err := binary.Read(deframer, binary.LittleEndian, &msgtype); err != nil {
				log.Fatal(err)
			}
			// TODO: rewrite this mess
			switch msgtype {
			case protocol.SetDeltaTime:
				// TODO: add a method on the Session to set delta time or
				// whatever?
				binary.Read(deframer, binary.LittleEndian, &s.Δt)

			case protocol.ResetWorld:
				var maxEntities int64
				binary.Read(deframer, binary.LittleEndian, &maxEntities)
				// TODO: for client-only entities we can use the high
				// indices that server won't use. For this we'll want to
				// change IDAlloc to be aware of two heaps, one primary
				// (indices 0 to server's entity limit) and one
				// secondary (above the primary)
				log.Print("maxEntities=", maxEntities)
				s.mu.Lock()
				s.world = worldspawn.NewWorld(int(maxEntities))
				// TODO: we should also stop rendering until we get the first
				// UpdateWorld
				s.mu.Unlock()

				// The renderer will resize its scene by itself

			case protocol.SetPlayer:
				var player ecs.ID
				binary.Read(deframer, binary.LittleEndian, &player)
				s.mu.Lock()
				s.player = player
				s.mu.Unlock()

			case protocol.UpdateWorld:
				// TODO: be careful to only read up to a limit.
				buf, err := io.ReadAll(deframer)
				if err != nil {
					log.Fatal("hmm2 ", err)
				}

				if err := s.handleUpdate(buf, slog.Default()); err != nil {
					log.Fatal("failed to handle server snapshot: ", err)
				}

				select {
				case ready <- struct{}{}:
					log.Println("received the first snapshot")
				default:
				}

			default:
				slog.Warn("unknown message type", "msgtype", msgtype)
			}

			n, _ := io.Copy(io.Discard, deframer)
			if n != 0 {
				slog.Warn("trailing bytes", "msgtype", msgtype, "bytes", []byte{}) // TODO: dump the bytes
			}

			if err := deframer.Next(); err != nil {
				log.Fatal("err=", err)
			}
		}
	}()
	<-ready

	go func() {
		ticker := time.NewTicker(s.Δt)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
			case <-s.done:
				return
			}

			s.tick(s.Δt)
		}
	}()

	return s, nil
}

func (s *Client) handleUpdate(buf []byte, logger *slog.Logger) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	r := bytes.NewReader(buf)

	dec := nice.NewDecoder(r, nice.WithUnmarshaler(worldspawn.EntityNiceUnmarshaler))

	if err := nice.UnmarshalDecode(dec, &s.world.SingletonComponents); err != nil {
		return err
	}

	{
		var n int
		if err := nice.UnmarshalDecode(dec, &n); err != nil {
			return err
		}
		for range n {
			var indexGen struct{ Index, Gen uint32 }
			if err := nice.UnmarshalDecode(dec, &indexGen); err != nil {
				return err
			}

			if indexGen.Gen != ^uint32(0) {
				id := ecs.MakeID(int(indexGen.Index), indexGen.Gen)

				idAtIndex := s.world.IDAlloc.Index(int(indexGen.Index))
				if idAtIndex != id {
					if idAtIndex != 0 {
						// TODO: log that we're deleting this
						s.world.DeleteEntityImmediately(idAtIndex)
					}
					if err := s.world.IDAlloc.AllocAt(id); err != nil {
						panic(err) // TODO: handle properly
					}
					// TODO: not sure if slog.Info or slog.Debug?
					// TODO: rework these slog messages
					logger.Debug("create entity", "index", indexGen.Index, "id", id, "replaces", idAtIndex)
				}
			} else {
				id := s.world.IDAlloc.Index(int(indexGen.Index))
				if id != 0 {
					s.world.DeleteEntityImmediately(id)
					logger.Debug("delete entity", "index", indexGen.Index, "id", id)
				}
				// TODO: do slog.Warn if trying to delete an entity that isn't there
			}
		}
	}

	// TODO: clean this horrible mess up

	// TODO: see server/main.go for how we should deal with
	// this.
	comps := []string{
		"Entity",
		"TranslationRotation",
		"Scale",
		"RendererModel",
		"SoundEffect",
		"CosmeticOffset",
		"Animation", // stress test map nice un/marshaling
		"Pose",      // stress test map nice un/marshaling
		"PhysicsShape",
		"PhysicsLayer",
		"PhysicsMotionType",
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

	for _, comp := range comps {
		cs := ecs.Reflect(reflect.ValueOf(&s.world.Components).Elem().FieldByName(comp).Addr().Interface())

		v := reflect.New(cs.ElemType())

		var n int
		if err := nice.UnmarshalDecode(dec, &n); err != nil {
			return err
		}
		for range n {
			// TODO: rename
			var indexAndOk uint32
			if err := nice.UnmarshalDecode(dec, &indexAndOk); err != nil {
				return err
			}
			index := indexAndOk &^ (1 << 31)
			exists := indexAndOk&(1<<31) != 0
			if exists {
				if err := nice.UnmarshalDecode(dec, v.Interface()); err != nil {
					return err
				}
			}

			id := s.world.IDAlloc.Index(int(index))
			if id == 0 {
				// This usually indicates a bug in the game code.
				logger.Warn("entity does not exist at an index", "component", comp, "index", index)
				continue
			}

			if exists {
				cs.Store(id, v.Elem())
			} else {
				cs.Delete(id)
			}
		}
	}

	return nil
}

func (s *Client) HandleInput(cmds []worldspawn.TimestampedInputCmd) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.inputCmds = append(s.inputCmds, cmds...)

	for _, cmd := range cmds {
		s.world.HandleInput(s.player, cmd, s.Δt, worldspawn.UpdateSpeculative, nil)
	}
	s.renderer.Subtick(s.world, s.player)
}

func (s *Client) tick(Δt time.Duration) {
	defer trace.StartRegion(context.Background(), "Tick").End()

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.inputCmds) > 0 {
		buf := new(bytes.Buffer)

		if err := writeInputCmds(buf, s.inputCmds); err != nil {
			panic(err)
		}
		s.inputCmds = s.inputCmds[:0]

		// TODO: this can block which is not very nice... or maybe that's fine?
		msg := protocol.NewFramer(s.inputStream)
		msg.Write(buf.Bytes())
		msg.Next()
	}

	s.world.Update(Δt, worldspawn.UpdateSpeculative, slog.Default())

	s.renderer.Tick(s.world, s.player, s.world.Now, s.world.Now.Add(Δt), Δt)

	worldspawn.ClearTransientComponents(s.world)

	s.world.Now = s.world.Now.Add(Δt)
}

// TODO: remove this func in favor of the caller just using nice directly?
func writeInputCmds(w io.Writer, cmds []worldspawn.TimestampedInputCmd) error {
	enc := nice.NewEncoder(w, nice.WithMarshaler(worldspawn.InputCommandNiceMarshal))
	return nice.MarshalEncode(enc, &cmds)
}

func (s *Client) Close() {
	close(s.done)
}

func durationSeconds(d time.Duration) float64 {
	return float64(d) / 1e9
}
