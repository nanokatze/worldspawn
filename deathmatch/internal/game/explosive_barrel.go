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

			state := entity.ScriptState().(ExplosiveBarrel)
			T := world.GetGlobalTransform2(entity)

			if state.Health <= 0 {
				attacker := world.GetEntity2(state.Attacker)
				if !attacker.Valid() {
					// If there's nobody using us, report ourselves as the
					// attacker.
					attacker = entity
				}

				world.explosion(
					Impact{
						Attacker: attacker,
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

				io.EnqueueEntityUpdate(entity,
					func(info *UpdateParams, entity Entity2, io IO) {
						T := entity.Transform()
						entity.Clear()
						entity.SetScriptState(DeleteAfter{})
						entity.SetNextThink(info.Now.Add(2 * time.Second)) // TODO: should be long enough for sound to play
						entity.SetTransform(T)
						entity.SetSoundEffect(SoundEmitter{
							Effect:      "explosion.wav",
							Attenuation: 1,
							PlayTime:    info.Now.Add(info.Δt),
						})
					})
			}
		},
		Impact: func(info *UpdateParams, entity Entity2, impact Impact, io IO) {
			// TODO: verify impact preconditions here

			state := entity.ScriptState().(ExplosiveBarrel)
			defer func() { entity.SetScriptState(state) }()

			state.Attacker = impact.Attacker.ID()
			state.Health -= impact.Damage
		},
	}
}
