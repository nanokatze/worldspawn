package game

import (
	"math"
	"reflect"
	"time"

	"worldspawn/internal/ecs"
)

// TODO: projectiles should not rely on the physics engine to figure out
// contacts but on queries instead, at least for now. In the future, if we
// change the player's collider to a dynamic physics object (rather than rolling
// our own physics) and our physics engine had a way to establish rules around
// energy transfer, we could switch back to using physics engine for contacts.

type GrenadeInFlight struct {
	LaunchedAt Time // when the fuse was ignited
	ExplodeNow bool // whether we should explode ASAP

	Attacker ecs.ID // who to attribute this damage to
}

func (GrenadeInFlight) entity() {}

func init() {
	scripts[reflect.TypeFor[GrenadeInFlight]()] = script{
		Think: func(info *UpdateParams, world *World, grenade ecs.ID, io IO) {
			const fuse = 1400 * time.Millisecond

			grenadeState, _ := world.GetEntity[GrenadeInFlight](grenade)
			if grenadeState.LaunchedAt.Add(fuse).After(io.world.Now) && !grenadeState.ExplodeNow {
				return
			}

			radialImpact(
				world,
				Impact{
					Attacker: grenadeState.Attacker,
					Type:     BlastImpactWithFragmentation,
					Damage:   1500,
				},
				world.GetGlobalTransform(grenade),
				sphericalExplosion,
				5,
				4*math.Pi/500,
				QueryFilters{},
				io)

			io.EnqueueEntityUpdate(grenade,
				func(info *UpdateParams, grenade ecs.ID, io IO) {
					T := io.world.GetTransform(grenade)
					io.world.ClearEntity(grenade)
					io.world.Entity.Set(grenade, DeleteAfter{})
					io.world.NextThink.Set(grenade, io.world.Now.Add(2*time.Second))
					io.world.SetTransform(grenade, T)
					io.world.SoundEffect.Set(grenade, SoundEmitter{
						Effect:      "explosion.wav",
						Attenuation: 1,
						PlayTime:    io.world.Now.Add(info.Δt),
					})
				})
		},

		ContactAdded: func(info *UpdateParams, world *World, grenade, entity2 ecs.ID) {
			grenadeState, _ := world.GetEntity[GrenadeInFlight](grenade)
			if entity2 == grenadeState.Attacker {
				// TODO: this should not be reachable, but for now it is. We
				// should ignore this contact in ShouldCollide.
				return
			}

			if _, ok := world.CosmeticOffset.Get(grenade); ok {
				world.CosmeticOffset.Delete(grenade)
			}

			if _, ok := world.ShouldSetOffFuseOnImpact.Get(entity2); ok {
				grenadeState.ExplodeNow = true
				world.Entity.Set(grenade, grenadeState)
			}
		},
	}
}
