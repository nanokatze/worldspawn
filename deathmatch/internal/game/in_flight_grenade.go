package game

import (
	"math"
	"time"

	"worldspawn/internal/ecs"
)

func init() {
	scripts["in_flight_grenade"] = scriptFuncs{
		Think: func(world *World, grenade ecs.ID, info *UpdateParams) {
			world.radialImpact(
				Impact{
					Inflictor: grenade, // TODO: this should be the character, actually
					Type:      BlastImpactWithFragmentation,
					Damage:    1500,
				},
				world.GetGlobalTransform(grenade),
				sphericalExplosion,
				5,
				4*math.Pi/500)

			world.EnqueueEntityUpdate(grenade,
				func(world *World, grenade ecs.ID, info *UpdateParams) {
					T := world.GetTransform(grenade)
					world.ClearEntity(grenade)
					world.SetScript(grenade, "delete_after")
					world.NextThink.Set(grenade, world.Now.Add(2*time.Second))
					world.SetTransform(grenade, T)
					world.SoundEffect.Set(grenade, SoundEmitter{
						Effect:      "explosion.wav",
						Attenuation: 1,
						PlayTime:    world.Now.Add(info.Δt),
					})
				})
		},
	}
}
