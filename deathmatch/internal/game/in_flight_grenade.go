package game

import (
	"math"
	"time"

	"worldspawn/internal/ecs"
)

func init() {
	scripts["in_flight_grenade"] = scriptFuncs{
		Think: func(scene *Scene, grenade ecs.ID, info *UpdateParams) {
			scene.radialImpact(
				Impact{
					Inflictor: grenade, // TODO: this should be the character, actually
					Type:      0,
					Damage:    1500,
				},
				scene.GetTransform(grenade).Compose(),
				sphericalExplosion, 3,
				4*math.Pi/500)

			scene.SendMessage(grenade,
				func(scene *Scene, grenade ecs.ID, info *UpdateParams) {
					T := scene.GetTransform(grenade)

					scene.ClearEntity(grenade)
					scene.SetScript(grenade, "delete_after")
					scene.NextThink.Set(grenade, scene.Now.Add(2*time.Second))
					scene.SetTransform(grenade, T)
					scene.SoundEffect.Set(grenade, SoundEmitter{
						Effect:      "explosion.wav",
						Attenuation: 1,
						PlayTime:    scene.Now.Add(info.Δt),
					})
				})
		},
	}
}
