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
		Think: func(info *UpdateParams, world *World, entity ecs.ID, io IO) {
			// TODO: we should implement this instead by spawning a pickup that
			// then gets picked up by the player. When that entity gets picked
			// up, we'll eventually respawn a new one.

			T := world.GetGlobalTransform(entity)

			for id, state := range ecs.All(&io.world.Entity) {
				if _, ok := state.(Gladiator); ok {
					playerT := io.world.GetGlobalTransform(id)

					if T.T.Sub(playerT.T).Length() <= 1.1 {
						io.EnqueueEntityUpdate(id,
							func(info *UpdateParams, id ecs.ID, io IO) {
								io.world.MutateEntity(id, func(v *Gladiator) { v.Inventory.Ammo[0] = 10 })
								info.Logger.Info("resupplied the player", "id", id)
							})

						io.EnqueueEntityUpdate(entity,
							func(info *UpdateParams, entity ecs.ID, io IO) {
								io.world.NextThink.Set(entity, io.world.Now.Add(10*time.Second))
							})

						break
					}
				}
			}
		},
	}
}
