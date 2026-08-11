package hud

import (
	"encoding/binary"
	"math"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/gpu"
	"worldspawn/gpu/image/draw"
	"worldspawn/gpu/vk"
)

// TODO: replace this with actual texture programs
var Progs = map[string]struct {
	Pack func(world *game.World, playerID game.EntityID, out []byte)
	Draw func(jq *gpu.JobQueue, dst *gpu.Image, data []byte)
}{
	"hud": {
		Pack: func(world *game.World, playerID game.EntityID, out []byte) {
			binary.LittleEndian.PutUint32(out[0:4], 0)

			player := world.Entity(playerID)
			if !player.IsValid() {
				return
			}
			playerState, ok := player.ScriptState().(game.Player)
			if !ok {
				return
			}

			pawn := world.Entity(playerState.Pawn)
			if !pawn.IsValid() {
				return
			}
			pawnState, ok := pawn.ScriptState().(game.Gladiator) // TODO: could we just poke a script function on the entity? Yeah we could I guess. Also on Player.
			if !ok {
				return
			}

			binary.LittleEndian.PutUint32(out[0:4], math.Float32bits(pawnState.Vitals.Health))
		},
		Draw: func(jq *gpu.JobQueue, dst *gpu.Image, data []byte) {
			// TODO: please actually draw like a well-adjusted person

			health := math.Float32frombits(binary.LittleEndian.Uint32(data))

			pass := draw.Begin(jq,
				&draw.Config{
					ColorAttachments: []draw.Attachment{
						{
							Image:  dst,
							LoadOp: vk.ATTACHMENT_LOAD_OP_CLEAR,
							ClearValue: [4]uint32{
								math.Float32bits(0.2),
								math.Float32bits(0),
								math.Float32bits(0),
								math.Float32bits(1),
							},
						},
					},
					RenderArea: vk.Rect2D{
						Offset: vk.Offset2D{X: 128, Y: 128},
						Extent: vk.Extent2D{Width: 1000, Height: 32},
					},
					LayerCount: 1,
				})
			pass.End()

			pass2 := draw.Begin(jq,
				&draw.Config{
					ColorAttachments: []draw.Attachment{
						{
							Image:  dst,
							LoadOp: vk.ATTACHMENT_LOAD_OP_CLEAR,
							ClearValue: [4]uint32{
								math.Float32bits(1),
								math.Float32bits(0),
								math.Float32bits(0),
								math.Float32bits(1),
							},
						},
					},
					RenderArea: vk.Rect2D{
						Offset: vk.Offset2D{X: 128, Y: 128},
						Extent: vk.Extent2D{Width: uint32(10 * max(health, 0)), Height: 32},
					},
					LayerCount: 1,
				})
			pass2.End()
		},
	},
}
