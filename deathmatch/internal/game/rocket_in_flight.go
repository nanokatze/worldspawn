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

func init() {
	Scripts[reflect.TypeFor[RocketInFlight]()] = script{
		Think: func(info *UpdateParams, world *World, rocket Entity2, io IO) {
			const fuse = 5000 * time.Millisecond

			rocketState := rocket.ScriptState().(RocketInFlight)
			if rocketState.LaunchedAt.Add(fuse).After(io.world.Now) && !rocketState.ExplodeNow {
				return
			}

			world.radialImpact(
				Impact{
					Attacker: rocketState.Attacker,
					Type:     BlastImpactWithFragmentation,
					Damage:   1200,
				},
				io.world.GetGlobalTransform2(rocket),
				sphericalExplosion,
				3,
				4*math.Pi/500,
				QueryFilters{
					Entity: func(id ecs.ID) bool { return id != rocket.ID() },
				},
				io)

			io.EnqueueEntityUpdate(rocket.ID(),
				func(info *UpdateParams, grenade Entity2, io IO) {
					T := grenade.Transform()
					grenade.Clear()
					grenade.SetScriptState(DeleteAfter{})
					grenade.SetNextThink(io.world.Now.Add(2 * time.Second)) // TODO: should be long enough for sound to play
					grenade.SetTransform(T)
					grenade.SetSoundEffect(SoundEmitter{
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
