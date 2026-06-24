package game

import (
	"math"
	"reflect"
	"time"

	"worldspawn/internal/ecs"
)

type RocketInFlight struct {
	LaunchedAt Time // when the fuse was ignited
	ExplodeNow bool // whether we should explode ASAP

	Attacker ecs.ID // who to attribute this damage to
}

func (RocketInFlight) entity() {}

func init() {
	scripts[reflect.TypeFor[RocketInFlight]()] = script{
		Think: func(world *World, grenade ecs.ID, info *UpdateParams) {
			io := IO{world.Updates, &world.globalUpdates, grenade}

			const fuse = 5000 * time.Millisecond

			rocketState, _ := world.GetEntity[RocketInFlight](grenade)
			if rocketState.LaunchedAt.Add(fuse).After(world.Now) && !rocketState.ExplodeNow {
				return
			}

			radialImpact(
				world,
				&io,
				Impact{
					Attacker: rocketState.Attacker,
					Type:     BlastImpactWithFragmentation,
					Damage:   1200,
				},
				world.GetGlobalTransform(grenade),
				sphericalExplosion,
				3,
				4*math.Pi/500,
				QueryFilters{})

			io.EnqueueEntityUpdate(grenade,
				func(world *World, grenade ecs.ID, info *UpdateParams) {
					T := world.GetTransform(grenade)
					world.ClearEntity(grenade)
					world.Entity.Set(grenade, DeleteAfter{})
					world.NextThink.Set(grenade, world.Now.Add(2*time.Second)) // TODO: should be long enough for sound to play
					world.SetTransform(grenade, T)
					world.SoundEffect.Set(grenade, SoundEmitter{
						Effect:      "explosion.wav",
						Attenuation: 1,
						PlayTime:    world.Now.Add(info.Δt),
					})
				})
		},

		ContactAdded: func(world *World, grenade, entity2 ecs.ID, _ *UpdateParams) {
			rocketState, _ := world.GetEntity[RocketInFlight](grenade)
			if entity2 == rocketState.Attacker {
				// TODO: filter this in ShouldCollide. This should never be reached
				return
			}

			rocketState.ExplodeNow = true
			world.Entity.Set(grenade, rocketState)
		},
	}
}
