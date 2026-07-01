package game

import (
	"reflect"
	"time"

	"worldspawn/internal/ecs"
)

type AmmoPickup struct{}

func (AmmoPickup) entity() {}

func init() {
	scripts[reflect.TypeFor[AmmoPickup]()] = script{
		Think: func(world *World, entity ecs.ID, info *UpdateParams) {
			// TODO: we should implement this instead by spawning a pickup that
			// then gets picked up by the player. When that entity gets picked
			// up, we'll eventually respawn a new one.

			io := IO{world.Updates, &world.globalUpdates, entity}

			T := world.GetGlobalTransform(entity)

			for id, state := range ecs.All(&world.Entity) {
				if _, ok := state.(Gladiator); ok {
					playerT := world.GetGlobalTransform(id)

					if T.T.Sub(playerT.T).Length() <= 1.1 {
						io.EnqueueEntityUpdate(id,
							func(world *World, id ecs.ID, info *UpdateParams) {
								gladiatorState, _ := world.GetEntity[Gladiator](id)
								gladiatorState.Inventory.Ammo[0] = 10
								world.Entity.Set(id, gladiatorState)
								info.Logger.Info("resupplied the player", "id", id)
							})

						world.NextThink.Set(entity, world.Now.Add(10*time.Second))

						break
					}
				}
			}
		},
	}
}
