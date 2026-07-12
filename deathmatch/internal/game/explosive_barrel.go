package game

import (
	"math"
	"reflect"
	"time"

	"worldspawn/internal/ecs"
)

// TODO: rename to something better, e.g. ExplodingProp
type ExplosiveBarrel struct {
	// TODO:

	Health int32

	Attacker ecs.ID // who is using us to cause damage
}

func init() {
	Scripts[reflect.TypeFor[ExplosiveBarrel]()] = script{
		Think: func(info *UpdateParams, world *World, entity Entity2, io IO) {
			// TODO: we should do a SetNextThink to forever and have Impact
			// SetNextThink asap otherwise

			barrel := entity.ScriptState().(ExplosiveBarrel)
			T := world.GetGlobalTransform2(entity)

			if barrel.Health <= 0 {
				world.radialImpact(
					Impact{
						Attacker: entity.ID(),                  // TODO: we'll want to track who shot/punted us
						Type:     BlastImpactWithFragmentation, // TODO: we should specify impact type and damage on the barrel itself I think
						Damage:   1500,
					},
					T,
					sphericalExplosion,
					5,
					4*math.Pi/500,
					QueryFilters{
						Entity: func(id ecs.ID) bool {
							return id != entity.ID()
						},
					},
					io)

				io.EnqueueEntityUpdate(entity.ID(),
					func(info *UpdateParams, entity Entity2, io IO) {
						T := entity.Transform()
						entity.Clear()
						entity.SetScriptState(DeleteAfter{})
						entity.SetNextThink(io.world.Now.Add(2 * time.Second)) // TODO: should be long enough for sound to play
						entity.SetTransform(T)
						entity.SetSoundEffect(SoundEmitter{
							Effect:      "explosion.wav",
							Attenuation: 1,
							PlayTime:    io.world.Now.Add(info.Δt),
						})
					})
			}
		},
		Impact: func(info *UpdateParams, entity Entity2, impact Impact, io IO) {
			// TODO: verify impact preconditions here

			barrel := entity.ScriptState().(ExplosiveBarrel)
			defer func() { entity.SetScriptState(barrel) }()

			barrel.Attacker = impact.Attacker
			barrel.Health -= impact.Damage
		},
	}
}
