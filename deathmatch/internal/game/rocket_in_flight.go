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
		Think: func(info *UpdateParams, world *World, grenade ecs.ID, io IO) {
			const fuse = 5000 * time.Millisecond

			rocketState, _ := io.world.GetEntity[RocketInFlight](grenade)
			if rocketState.LaunchedAt.Add(fuse).After(io.world.Now) && !rocketState.ExplodeNow {
				return
			}

			radialImpact(
				world,
				Impact{
					Attacker: rocketState.Attacker,
					Type:     BlastImpactWithFragmentation,
					Damage:   1200,
				},
				io.world.GetGlobalTransform(grenade),
				sphericalExplosion,
				3,
				4*math.Pi/500,
				QueryFilters{},
				io)

			io.EnqueueEntityUpdate(grenade,
				func(info *UpdateParams, grenade ecs.ID, io IO) {
					T := io.world.GetTransform(grenade)
					io.world.ClearEntity(grenade)
					io.world.Entity.Set(grenade, DeleteAfter{})
					io.world.NextThink.Set(grenade, io.world.Now.Add(2*time.Second)) // TODO: should be long enough for sound to play
					io.world.SetTransform(grenade, T)
					io.world.SoundEffect.Set(grenade, SoundEmitter{
						Effect:      "explosion.wav",
						Attenuation: 1,
						PlayTime:    io.world.Now.Add(info.Δt),
					})
				})
		},

		ContactAdded: func(_ *UpdateParams, world *World, grenade, entity2 ecs.ID) {
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
