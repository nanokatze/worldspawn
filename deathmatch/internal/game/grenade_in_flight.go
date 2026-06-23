package game

import (
	"math"
	"reflect"
	"time"

	"worldspawn/internal/ecs"
)

type GrenadeInFlight struct {
	LaunchedAt Time // when the fuse was ignited
	ExplodeNow bool // whether we should explode ASAP

	Attacker ecs.ID // who to attribute this damage to
}

func (GrenadeInFlight) entity() {}

func init() {
	scripts["grenade_in_flight"] = scriptFuncs{
		Types: []reflect.Type{reflect.TypeFor[GrenadeInFlight]()},

		Think: func(world *World, grenade ecs.ID, info *UpdateParams) {
			const fuse = 1400 * time.Millisecond

			grenadeState, _ := world.GetEntity[GrenadeInFlight](grenade)
			if grenadeState.LaunchedAt.Add(fuse).After(world.Now) && !grenadeState.ExplodeNow {
				return
			}

			world.radialImpact(
				Impact{
					Attacker: grenadeState.Attacker,
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

		ContactAdded: func(world *World, grenade, entity2 ecs.ID, info *UpdateParams) {
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
