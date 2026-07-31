package game

import (
	"reflect"
	"time"

	"worldspawn/internal/ecs"
)

type AmmoPickup struct{}

func init() {
	Scripts[reflect.TypeFor[AmmoPickup]()] = script{
		Think: func(stx ScriptContext, world *World, entity Entity2) {
			// TODO: we should implement this instead by spawning a pickup that
			// then gets picked up by the player. When that entity gets picked
			// up, we'll eventually respawn a new one.

			T := world.GetGlobalTransform2(entity)

			for id, state := range ecs.All(&world.ScriptState) {
				if _, ok := state.(Gladiator); ok {
					player := world.GetEntity2(id)
					playerT := world.GetGlobalTransform(id)

					if T.T.Sub(playerT.T).Length() <= 1.1 {
						stx.Update(player,
							func(stx ScriptContext, player Entity2) {
								state := player.ScriptState().(Gladiator)
								defer func() { player.SetScriptState(state) }()

								state.Inventory.Ammo[0] = 10
								state.Inventory.Ammo[1] = 100

								entity.Logger().Info("resupplied", "player", player.ID())
							})

						stx.Update(entity,
							func(stx ScriptContext, entity Entity2) {
								entity.SetNextThink(stx.Now.Add(10 * time.Second))
							})

						break
					}
				}
			}
		},
	}
}
