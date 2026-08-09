package game

import (
	"math"
	"reflect"
	"time"
	"unique"

	"worldspawn/internal/ecs"
)

type RocketInFlight struct {
	Attacker   ecs.ID // who to attribute this damage to
	LaunchedAt Time   // when the fuse was ignited

	ExplodeNow bool // whether we should explode now
}

func init() {
	Scripts[reflect.TypeFor[RocketInFlight]()] = script{
		Think: func(stx ScriptContext, world *World, rocket Entity) {
			const fuse = 5000 * time.Millisecond

			state := rocket.ScriptState().(RocketInFlight)
			if state.LaunchedAt.Add(fuse).Compare(stx.Now) > 0 && !state.ExplodeNow {
				return
			}

			world.explosion(
				stx,
				Impact{
					Attacker: world.Entity(state.Attacker),
					Type:     BlastImpactWithFragmentation,
					Damage:   1200,
				},
				world.GetGlobalTransform2(rocket),
				sphericalExplosion,
				3,
				4*math.Pi/500,
				QueryFilters{
					Entity: func(id ecs.ID) bool { return id != rocket.ID() },
				})

			stx.Update(rocket,
				func(stx ScriptContext, rocket Entity) {
					T := rocket.Transform()
					rocket.Clear()
					rocket.SetScriptState(DeleteAfter{})
					rocket.SetNextThink(stx.Now.Add(2 * time.Second)) // TODO: should be long enough for sound to play
					rocket.SetTransform(T)
					rocket.SetSoundEffect(SoundEmitter{
						Effect:      unique.Make("common/sounds/Explosion"),
						Attenuation: 1,
						PlayTime:    stx.Now.Add(stx.Δt),
					})
				})
		},

		ContactAdded: func(stx ScriptContext, world *World, rocket, entity2 ecs.ID) {
			state := world.Entity(rocket).ScriptState().(RocketInFlight)
			if entity2 == state.Attacker {
				// TODO: filter this in ShouldCollide. This should never be reached
				return
			}
			defer func() { world.Entity(rocket).SetScriptState(state) }()

			state.ExplodeNow = true
		},
	}
}
