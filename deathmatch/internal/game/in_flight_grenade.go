package game

import (
	"math"
	"time"
	"worldspawn/internal/ecs"
)

var grenadeStats = struct {
	ExplosionSound string

	FuseDuration time.Duration `json:",format:iso8601"`
}{
	ExplosionSound: "explosion.wav",

	FuseDuration: 1400 * time.Millisecond,
}

func init() {
	scripts["in_flight_grenade"] = scriptFuncs{
		Think: func(scene *Scene, grenade ecs.ID, info *UpdateParams) {
			scene.radialImpact(
				Impact{
					Type:      0,
					Damage:    300,
					Inflictor: grenade, // TODO: this should be the character, actually
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
						Effect:      grenadeStats.ExplosionSound,
						Attenuation: 1,
						PlayTime:    scene.Now.Add(info.Δt),
					})
				})
		},
	}
}
