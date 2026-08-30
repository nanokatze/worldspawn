package game

import (
	"math"
	"reflect"
	"time"
	"unique"
)

// TODO: rename to something better, e.g. ExplodingProp
type ExplosiveBarrel struct {
	// TODO:

	Health float32

	Attacker EntityID // who is using us to cause damage
}

func init() {
	Scripts[reflect.TypeFor[ExplosiveBarrel]()] = script{
		Think: func(stx ScriptContext, entity Entity, world *World) {
			// TODO: we should do a SetNextThink to forever and have Impact
			// SetNextThink asap otherwise

			state := entity.ScriptState().(ExplosiveBarrel)

			if state.Health <= 0 {
				T := world.GetGlobalTransform2(entity)

				attacker := world.Entity(state.Attacker)
				if !attacker.IsValid() {
					// If there's nobody using us, report ourselves as the
					// attacker.
					attacker = entity
				}

				world.explosion(
					stx,
					Impact{
						Attacker: attacker,
						Type:     ImpactTypeBlast, // TODO: we should specify impact type and damage on the barrel itself I think
						Damage:   1500,
					},
					T,
					sphericalExplosion,
					5,
					4*math.Pi/500,
					QueryFilters{
						Entity: func(id EntityID) bool {
							return id != entity.ID()
						},
					})

				stx.Update(entity,
					func(stx ScriptContext, entity Entity) {
						T := entity.Transform()
						entity.Clear()
						entity.SetScriptState(DeleteAfter{})
						entity.SetNextThink(stx.Now.Add(2 * time.Second)) // TODO: should be long enough for sound to play
						entity.SetTransform(T)
						entity.SetSoundEffect(SoundEmitter{
							Effect:      unique.Make("common/sounds/Explosion"),
							Attenuation: 1,
							PlayTime:    stx.Now.Add(stx.Δt),
						})
					})
			}
		},
		HandleImpact: func(stx ScriptContext, entity Entity, impact Impact) {
			// TODO: verify impact preconditions here

			state := entity.ScriptState().(ExplosiveBarrel)
			defer func() { entity.SetScriptState(state) }()

			state.Attacker = impact.Attacker.ID()
			state.Health -= impact.Damage
		},
	}
}
