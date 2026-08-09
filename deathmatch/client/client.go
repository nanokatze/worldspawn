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
	"worldspawn/internal/netutil"
	"worldspawn/internal/nice"
)

// TODO: move client to deathmatch/internal so we can have a separate bot client
// program that way? This Renderer interface would be moved there as well.

type Renderer interface {
	Reset(n int)
	// TODO: stop using game.Time here. We'll first need to address how our
	// sounds work though.
	Update(world *game.World, playerID ecs.ID, t0, t1 game.Time, frameDuration time.Duration)
	UpdateSubtick(world *game.World, playerID ecs.ID)
}

type Client struct {
	done chan struct{} // TODO: rename to something like "close"

	mu sync.Mutex

	conn        *quic.Conn
	inputStream *quic.SendStream

	inputCmds []game.TimestampedInputCmd

	tickPeriod time.Duration
	world      *game.World

	player ecs.ID

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
		deframer := netutil.NewDeframer(stream2)
		for {
			var msgtype uint64
			if err := binary.Read(deframer, binary.LittleEndian, &msgtype); err != nil {
				log.Fatal(err)
			}

			switch msgtype {
			case replication.ResetTicker:
				// TODO: we should restart the ticker with the new period when this happens
				binary.Read(deframer, binary.LittleEndian, &s.tickPeriod)

			case replication.ResetWorld:
				var cap int64
				binary.Read(deframer, binary.LittleEndian, &cap)
				// TODO: for client-only entities we can use the high
				// indices that server won't use. For this we'll want to
				// change IDAlloc to be aware of two heaps, one primary
				// (indices 0 to server's entity limit) and one
				// secondary (above the primary)
				slog.Info("world reset", "cap", cap)
				s.mu.Lock()
				s.world = new(game.World)
				s.world.Reset(int(cap))
				// TODO: we should also stop rendering until we get the first
				// UpdateWorld
				s.renderer.Reset(int(cap))
				s.mu.Unlock()

				// The renderer will resize its scene by itself

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

			case replication.SetPlayer:
				var player ecs.ID
				binary.Read(deframer, binary.LittleEndian, &player)
				s.mu.Lock()
				s.player = player
				s.mu.Unlock()

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
		ticker := time.NewTicker(s.tickPeriod)

		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
			}

			s.tick(s.tickPeriod)
		}
	}()

	return s, nil
}

func (s *Client) handleUpdate(buf []byte, logger *slog.Logger) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	r := bytes.NewReader(buf)

	opts := replication.NiceOptions2(s.world)

	dec := nice.NewDecoder(r, opts)

	{
		var n int
		if err := nice.UnmarshalDecode(dec, &n); err != nil {
			return err
		}
		for range n {
			var indexGen struct{ Index, Gen uint32 } // TODO: replace with plain id?
			if err := nice.UnmarshalDecode(dec, &indexGen); err != nil {
				return err
			}

			if indexGen.Gen != ^uint32(0) {
				id := ecs.MakeID(int(indexGen.Index), indexGen.Gen)

				idAtIndex := s.world.Table.IDs().Index(int(indexGen.Index))
				if idAtIndex != id {
					if idAtIndex != 0 {
						// TODO: log that we're deleting this
						s.world.DeleteEntityImmediately(idAtIndex)
					}
					if !s.world.Table.CreateRow(id) {
						panic("bad") // TODO: handle properly
					}
					// TODO: not sure if slog.Info or slog.Debug?
					// TODO: rework these slog messages
					logger.Debug("create entity", "index", indexGen.Index, "id", id, "replaces", idAtIndex)
				}
			} else {
				id := s.world.Table.IDs().Index(int(indexGen.Index))
				if id != 0 {
					// In this particular case, we can just mark it for deletion
					// and delete in a pass later.
					s.world.DeleteEntityImmediately(id)
					logger.Debug("delete entity", "index", indexGen.Index, "id", id)
				}
				// TODO: do slog.Warn if trying to delete an entity that isn't there
			}
		}
	}

	// TODO: clean this horrible mess up

	for _, columnIndex := range game.ReplicatedColumns {
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
			index := indexAndOk & ((1 << 31) - 1)
			exists := indexAndOk&(1<<31) != 0
			if exists {
				if err := nice.UnmarshalDecode(dec, v.Interface()); err != nil {
					return err
				}
			}

			id := s.world.Table.IDs().Index(int(index))
			if id == 0 {
				// TODO: figure out how to get the time
				logger.Warn("snapshot refers to a non-existent object", "t", game.Time{}, "index", index, "column", reflect.TypeFor[game.Columns]().Field(columnIndex).Name)
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
		s.world.HandleInput(s.player, cmd, &game.UpdateParams{Δt: s.tickPeriod, Speculating: true})
	}
	s.renderer.UpdateSubtick(s.world, s.player)
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
		msg := netutil.NewFramer(s.inputStream)
		msg.Write(buf.Bytes())
		msg.Next()
	}

	t0 := s.world.Entity(1).ScriptState().(game.Worldspawn).Now
	s.world.Step(game.UpdateParams{Δt: Δt, Speculating: true})
	s.renderer.Update(s.world, s.player, t0, s.world.Entity(1).ScriptState().(game.Worldspawn).Now, Δt)
}

// TODO: remove this func in favor of the caller just using nice directly?
func writeInputCmds(w io.Writer, cmds []game.TimestampedInputCmd) error {
	enc := nice.NewEncoder(w, replication.NiceOptions3)
	return nice.MarshalEncode(enc, &cmds)
}

func (s *Client) Close() {
	close(s.done)
}
