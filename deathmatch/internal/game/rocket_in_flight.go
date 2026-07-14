package game

import (
	"math"
	"reflect"
	"time"

	"worldspawn/internal/ecs"
)

type RocketInFlight struct {
	Attacker   ecs.ID // who to attribute this damage to
	LaunchedAt Time   // when the fuse was ignited

	ExplodeNow bool // whether we should explode now
}

func init() {
	Scripts[reflect.TypeFor[RocketInFlight]()] = script{
		Think: func(info *UpdateParams, world *World, rocket Entity2, io IO) {
			const fuse = 5000 * time.Millisecond

			state := rocket.ScriptState().(RocketInFlight)
			if state.LaunchedAt.Add(fuse).After(info.Now) && !state.ExplodeNow {
				return
			}

			world.explosion(
				Impact{
					Attacker: state.Attacker,
					Type:     BlastImpactWithFragmentation,
					Damage:   1200,
				},
				world.GetGlobalTransform2(rocket),
				sphericalExplosion,
				3,
				4*math.Pi/500,
				QueryFilters{
					Entity: func(id ecs.ID) bool { return id != rocket.ID() },
				},
				io)

			io.EnqueueEntityUpdate(rocket,
				func(info *UpdateParams, rocket Entity2, io IO) {
					T := rocket.Transform()
					rocket.Clear()
					rocket.SetScriptState(DeleteAfter{})
					rocket.SetNextThink(info.Now.Add(2 * time.Second)) // TODO: should be long enough for sound to play
					rocket.SetTransform(T)
					rocket.SetSoundEffect(SoundEmitter{
						Effect:      "explosion.wav",
						Attenuation: 1,
						PlayTime:    info.Now.Add(info.Δt),
					})
				})
		},

		ContactAdded: func(_ *UpdateParams, world *World, grenade, entity2 ecs.ID) {
			state, _ := world.GetEntity[RocketInFlight](grenade)
			if entity2 == state.Attacker {
				// TODO: filter this in ShouldCollide. This should never be reached
				return
			}
			defer func() { world.Entity.Set(grenade, state) }()

			state.ExplodeNow = true
		},
	}
}
