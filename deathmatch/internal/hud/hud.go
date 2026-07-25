// TODO: move this into the game code perhaps?
package hud

import (
	"math"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/gpu"
	"worldspawn/gpu/image/draw"
	"worldspawn/gpu/vk"
	"worldspawn/internal/ecs"
)

// This is basically root of the scene (widget tree.) We'll want to figure out
// how to do animations. We'll do a widget tree like gio UI does.
type State struct {
	Health int32
}

func (state *State) Update(world *game.World, playerID ecs.ID) {
	state.Health = 0

	player := world.GetEntity2(playerID)
	if !player.Valid() {
		return
	}
	playerState, ok := player.ScriptState().(game.Player)
	if !ok {
		return
	}

	pawn := world.GetEntity2(playerState.Pawn)
	if !pawn.Valid() {
		return
	}
	pawnState, ok := pawn.ScriptState().(game.Gladiator) // TODO: could we just poke a script function on the entity? Yeah we could I guess. Also on Player.
	if !ok {
		return
	}

	state.Health = max(pawnState.Vitals.Health, 0)
}

// TODO: it would be nice if this fed into some kind of vector rasterizer
// instead.
// TODO: we also need to pass scale. And we probably want to pass a subrect of
// dst image.
func (state *State) Draw(jq *gpu.JobQueue, dst *gpu.Image) {
	// TODO: please actually draw like a well-adjusted person

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
				Extent: vk.Extent2D{Width: uint32(10 * max(state.Health, 0)), Height: 32},
			},
			LayerCount: 1,
		})
	pass2.End()
}
