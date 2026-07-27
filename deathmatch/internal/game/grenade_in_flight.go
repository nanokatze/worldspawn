package game

import (
	"math"
	"reflect"
	"time"
	"unique"

	"worldspawn/internal/ecs"
)

// TODO: projectiles should not rely on the physics engine to figure out
// contacts but on queries instead, at least for now. In the future, if we
// change the player's collider to a dynamic physics object (rather than rolling
// our own physics) and our physics engine had a way to establish rules around
// energy transfer, we could switch back to using physics engine for contacts.

type GrenadeInFlight struct {
	Attacker   ecs.ID // who to attribute the damage to
	LaunchedAt Time   // when the fuse was ignited

	ExplodeNow bool // whether we should explode now
}

func init() {
	Scripts[reflect.TypeFor[GrenadeInFlight]()] = script{
		Think: func(info *UpdateParams, world *World, grenade Entity2, io IO) {
			const fuse = 1400 * time.Millisecond

			state := grenade.ScriptState().(GrenadeInFlight)
			if state.LaunchedAt.Add(fuse).After(info.Now) && !state.ExplodeNow {
				return
			}

			world.explosion(
				Impact{
					Attacker: world.GetEntity2(state.Attacker),
					Type:     BlastImpactWithFragmentation,
					Damage:   1500,
				},
				world.GetGlobalTransform2(grenade),
				sphericalExplosion,
				5,
				4*math.Pi/500,
				QueryFilters{
					Entity: func(id ecs.ID) bool { return id != grenade.ID() },
				},
				io)

			// TODO: create a new entity instead?
			io.Update(grenade,
				func(info *UpdateParams, grenade Entity2, io IO) {
					T := grenade.Transform()
					grenade.Clear()
					grenade.SetScriptState(DeleteAfter{})
					grenade.SetNextThink(info.Now.Add(2 * time.Second))
					grenade.SetTransform(T)
					grenade.SetSoundEffect(SoundEmitter{
						Effect:      unique.Make("explosion.wav"),
						Attenuation: 1,
						PlayTime:    info.Now.Add(info.Δt),
					})
				})
		},

		ContactAdded: func(info *UpdateParams, world *World, grenade, entity2 ecs.ID) {
			state := world.GetEntity2(grenade).ScriptState().(GrenadeInFlight)

			if entity2 == state.Attacker {
				// TODO: this should not be reachable, but for now it is. We
				// should ignore this contact in ShouldCollide.
				return
			}

			if _, ok := world.CosmeticOffset.Get(grenade); ok {
				world.CosmeticOffset.Delete(grenade)
			}

			if _, ok := world.ShouldSetOffFuseOnImpact.Get(entity2); ok {
				state.ExplodeNow = true
				world.GetEntity2(grenade).SetScriptState(state)
			}
		},
	}
}
