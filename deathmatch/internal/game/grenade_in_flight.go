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
		Think: func(stx ScriptContext, world *World, grenade Entity) {
			const fuse = 1400 * time.Millisecond

			state := grenade.ScriptState().(GrenadeInFlight)
			if state.LaunchedAt.Add(fuse).Compare(stx.Now) > 0 && !state.ExplodeNow {
				return
			}

			world.explosion(
				stx,
				Impact{
					Attacker: world.Entity(state.Attacker),
					Type:     BlastImpactWithFragmentation,
					Damage:   1500,
				},
				world.GetGlobalTransform2(grenade),
				sphericalExplosion,
				5,
				4*math.Pi/500,
				QueryFilters{
					Entity: func(id ecs.ID) bool { return id != grenade.ID() },
				})

			// TODO: create a new entity instead?
			stx.Update(grenade,
				func(stx ScriptContext, grenade Entity) {
					T := grenade.Transform()
					grenade.Clear()
					grenade.SetScriptState(DeleteAfter{})
					grenade.SetNextThink(stx.Now.Add(2 * time.Second))
					grenade.SetTransform(T)
					grenade.SetSoundEffect(SoundEmitter{
						Effect:      unique.Make("common/sounds/Explosion"),
						Attenuation: 1,
						PlayTime:    stx.Now.Add(stx.Δt),
					})
				})
		},

		ContactAdded: func(stx ScriptContext, world *World, grenade, entity2 ecs.ID) {
			state := world.Entity(grenade).ScriptState().(GrenadeInFlight)

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
				world.Entity(grenade).SetScriptState(state)
			}
		},
	}
}
