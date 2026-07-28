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
			info *UpdateParams,
			world *World,
			weapon Entity2,
			weaponProps []Entity2,
			attacker Entity2,
			T_attack gmath.Affine3f64,
			v_attack Velocity,
			buttons WeaponButtons,
			io IO) {
			state := weapon.ScriptState().(WeaponPhysgun)

			holdingEntity := world.GetEntity2(state.HeldEntity).Valid()
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

				if rayHit.Entity.Valid() {
					transform := T_attack.Inv().Mul(world.GetGlobalTransform(rayHit.Entity.ID()))

					io.Update(weapon,
						func(_ *UpdateParams, weapon Entity2, io IO) {
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

				io.Update(world.GetEntity2(state.HeldEntity),
					func(_ *UpdateParams, entity Entity2, io IO) {
						entity.SetTransform(transform.TRS())
						entity.SetVelocity(Velocity{})
					})

			case holdingEntity && !triggerHeld:
				io.Update(weapon,
					func(_ *UpdateParams, weapon Entity2, io IO) {
						state := weapon.ScriptState().(WeaponPhysgun)
						defer func() { weapon.SetScriptState(state) }()

						state.HeldEntity = ecs.NullID
					})
			}
		},
	}
}
