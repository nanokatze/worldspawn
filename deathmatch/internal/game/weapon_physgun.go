package game

import (
	"reflect"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
	"worldspawn/internal/physics"
)

type WeaponPhysgun struct {
	HeldEntity ecs.ID
	Transform  gmath.Affine3f64
}

func init() {
	Scripts[reflect.TypeFor[WeaponPhysgun]()] = script{
		Weapon_Hint: func(info *UpdateParams, world *World, weapon ecs.ID) WeaponHint {
			return WeaponHint{}
		},

		Weapon_Think: func(
			stx ScriptContext,
			world *World,
			weapon Entity,
			weaponProps []Entity,
			attacker Entity,
			T_attack gmath.Affine3f64,
			v_attack Velocity,
			buttons WeaponButtons) {
			state := weapon.ScriptState().(WeaponPhysgun)

			holdingEntity := world.Entity(state.HeldEntity).IsValid()
			triggerHeld := buttons&WeaponTrigger != 0

			switch {
			case !holdingEntity && triggerHeld:
				rayHit := world.TraceRay(
					physics.Ray{
						Origin:    T_attack.T,
						Direction: T_attack.M.Mulv(forward).Normalize(),
						TMax:      1000,
					},
					QueryFilters{
						Entity: func(entity ecs.ID) bool { return entity != attacker.ID() },
					})

				if rayHit.Entity.IsValid() {
					transform := T_attack.Inv().Mul(world.GetGlobalTransform2(rayHit.Entity))

					stx.Update(weapon,
						func(stx ScriptContext, weapon Entity) {
							state := weapon.ScriptState().(WeaponPhysgun)
							defer func() { weapon.SetScriptState(state) }()

							state.HeldEntity = rayHit.Entity.ID()
							state.Transform = transform
						})
				}

			case holdingEntity && triggerHeld:
				// TODO: this doesn't work correctly when we're touching an object
				// that's parented to something.
				transform := T_attack.Mul(state.Transform)

				stx.Update(world.Entity(state.HeldEntity),
					func(stx ScriptContext, entity Entity) {
						entity.SetTransform(transform.TRS())
						entity.SetVelocity(Velocity{})
					})

			case holdingEntity && !triggerHeld:
				stx.Update(weapon,
					func(stx ScriptContext, weapon Entity) {
						state := weapon.ScriptState().(WeaponPhysgun)
						defer func() { weapon.SetScriptState(state) }()

						state.HeldEntity = ecs.NullID
					})
			}
		},
	}
}
