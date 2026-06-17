package game

import (
	"math"
	"reflect"
	"time"

	"worldspawn/internal/ecs"
)

type InFlightGrenade struct {
	LaunchedAt Time // when the fuse was ignited
	ExplodeNow bool // whether we should explode ASAP

	Attacker ecs.ID // who to attribute this damage to
}

func (InFlightGrenade) entity() {}

func init() {
	scripts["in_flight_grenade"] = scriptFuncs{
		Types: []reflect.Type{reflect.TypeFor[InFlightGrenade]()},

		Think: func(world *World, grenade ecs.ID, info *UpdateParams) {
			const fuse = 1400 * time.Millisecond

			state, _ := world.GetEntity[InFlightGrenade](grenade)
			if state.LaunchedAt.Add(fuse).After(world.Now) && !state.ExplodeNow {
				return
			}

			world.radialImpact(
				Impact{
					Attacker: state.Attacker,
					Type:     BlastImpactWithFragmentation,
					Damage:   1500,
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
