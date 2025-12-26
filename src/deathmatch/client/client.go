package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"io"
	"log"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"github.com/quic-go/quic-go"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/deathmatch/internal/replication"
	"worldspawn/internal/ecs"
	"worldspawn/internal/framing"
	"worldspawn/internal/nice"
)

// TODO: move client to deathmatch/internal so we can have a separate bot client
// program that way? This Renderer interface would be moved there as well.

type Renderer interface {
	// TODO: rename to Update and possibly merge with Subtick somehow?
	Tick(w *game.Scene, camera ecs.Entity, t0, t1 game.Time, frameDuration time.Duration)
	Subtick(w *game.Scene, camera ecs.Entity)
}

type Client struct {
	done chan struct{} // TODO: rename to something like "close"

	mu sync.Mutex

	conn        *quic.Conn
	inputStream *quic.SendStream

	inputCmds []game.TimestampedInputCmd

	Δt    time.Duration
	world *game.Scene

	// TODO: we could be in control of many player entities
	player ecs.Entity

	renderer Renderer
}

// TODO: rename?
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
		deframer := framing.NewDeframer(stream2)
		for {
			var msgtype uint64
			if err := binary.Read(deframer, binary.LittleEndian, &msgtype); err != nil {
				log.Fatal(err)
			}
			// TODO: rewrite this mess
			switch msgtype {
			case replication.SetDeltaTime:
				// TODO: add a method on the Session to set delta time or
				// whatever?
				binary.Read(deframer, binary.LittleEndian, &s.Δt)

			case replication.ResetWorld:
				var maxEntities int64
				binary.Read(deframer, binary.LittleEndian, &maxEntities)
				// TODO: for client-only entities we can use the high
				// indices that server won't use. For this we'll want to
				// change IDAlloc to be aware of two heaps, one primary
				// (indices 0 to server's entity limit) and one
				// secondary (above the primary)
				log.Print("maxEntities=", maxEntities)
				s.mu.Lock()
				s.world = game.NewScene(int(maxEntities))
				// TODO: we should also stop rendering until we get the first
				// UpdateWorld
				s.mu.Unlock()

				// The renderer will resize its scene by itself

			case replication.SetPlayer:
				var player ecs.Entity
				binary.Read(deframer, binary.LittleEndian, &player)
				s.mu.Lock()
				s.player = player
				s.mu.Unlock()

			case replication.UpdateWorld:
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

	dec := nice.NewDecoder(r, replication.NiceOptions)

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
				id := ecs.MakeEntity(int(indexGen.Index), indexGen.Gen)

				idAtIndex := s.world.Entities.Index(int(indexGen.Index))
				if idAtIndex != id {
					if idAtIndex != 0 {
						// TODO: log that we're deleting this
						s.world.DeleteEntityImmediately(idAtIndex)
					}
					if err := s.world.Entities.AllocAt(id); err != nil {
						panic(err) // TODO: handle properly
					}
					// TODO: not sure if slog.Info or slog.Debug?
					// TODO: rework these slog messages
					logger.Debug("create entity", "index", indexGen.Index, "id", id, "replaces", idAtIndex)
				}
			} else {
				id := s.world.Entities.Index(int(indexGen.Index))
				if id != 0 {
					s.world.DeleteEntityImmediately(id)
					logger.Debug("delete entity", "index", indexGen.Index, "id", id)
				}
				// TODO: do slog.Warn if trying to delete an entity that isn't there
			}
		}
	}

	// TODO: clean this horrible mess up

	for columnIndex := range replication.Columns.NumField() {
		column := reflect.ValueOf(&s.world.Columns).Elem().Field(columnIndex).Addr().Interface().(ecs.AnyColumn).Reflect()

		v := reflect.New(column.ElemType())

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

			id := s.world.Entities.Index(int(index))
			if id == 0 {
				logger.Warn("entity does not exist at an index (likely a bug in the game code)", "component", replication.Columns.Field(columnIndex).Name, "index", index)
				continue
			}

			if exists {
				column.Set(id, v.Elem())
			} else {
				column.Delete(id)
			}
		}
	}

	return nil
}

func (s *Client) HandleInput(cmds []game.TimestampedInputCmd) {
	if len(cmds) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.inputCmds = append(s.inputCmds, cmds...)

	for _, cmd := range cmds {
		s.world.HandleInput(s.player, cmd, &game.UpdateParams{Δt: s.Δt, Speculating: true, Logger: slog.Default()})
	}
	s.renderer.Subtick(s.world, s.player)
}

func (s *Client) tick(Δt time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.inputCmds) > 0 {
		buf := new(bytes.Buffer)

		if err := writeInputCmds(buf, s.inputCmds); err != nil {
			panic(err)
		}
		s.inputCmds = s.inputCmds[:0]

		// TODO: this can block which is not very nice... or maybe that's fine?
		msg := framing.NewFramer(s.inputStream)
		msg.Write(buf.Bytes())
		msg.Next()
	}

	t0 := s.world.Now
	s.world.Update(&game.UpdateParams{Δt: Δt, Speculating: true, Logger: slog.Default()})
	s.renderer.Tick(s.world, s.player, t0, s.world.Now, Δt)

	game.ClearTransientComponents(s.world)
}

// TODO: remove this func in favor of the caller just using nice directly?
func writeInputCmds(w io.Writer, cmds []game.TimestampedInputCmd) error {
	enc := nice.NewEncoder(w, replication.NiceOptions)
	return nice.MarshalEncode(enc, &cmds)
}

func (s *Client) Close() {
	close(s.done)
}

func durationSeconds(d time.Duration) float64 {
	return float64(d) / 1e9
}
