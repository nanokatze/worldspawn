package game

import (
	"reflect"
	"time"

	"worldspawn/internal/ecs"
)

type AmmoPickup struct{}

func init() {
	Scripts[reflect.TypeFor[AmmoPickup]()] = script{
		Think: func(info *UpdateParams, world *World, entity Entity2, io IO) {
			// TODO: we should implement this instead by spawning a pickup that
			// then gets picked up by the player. When that entity gets picked
			// up, we'll eventually respawn a new one.

			T := world.GetGlobalTransform2(entity)

			for id, state := range ecs.All(&io.world.Entity) {
				if _, ok := state.(Gladiator); ok {
					playerT := io.world.GetGlobalTransform(id)

					if T.T.Sub(playerT.T).Length() <= 1.1 {
						io.EnqueueEntityUpdate(id,
							func(info *UpdateParams, entity Entity2, io IO) {
								state := entity.ScriptState().(Gladiator)
								defer func() { entity.SetScriptState(state) }()

								state.Inventory.Ammo[0] = 10
								state.Inventory.Ammo[1] = 100

								entity.Logger().Info("resupplied the player")
							})

						io.EnqueueEntityUpdate(entity.ID(),
							func(info *UpdateParams, entity Entity2, io IO) {
								entity.SetNextThink(info.Now.Add(10 * time.Second))
							})

						break
					}
				}
			}
		},
	}
}
