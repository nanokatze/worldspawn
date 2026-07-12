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
			weapon ecs.ID,
			props []ecs.ID,
			attacker ecs.ID,
			T_attack gmath.Affine3f64,
			v_attack Velocity,
			buttons WeaponButtons,
			io IO) Recoil {
			physgun, _ := world.GetEntity[WeaponPhysgun](weapon)

			holdingEntity := world.GetEntity2(physgun.HeldEntity).Valid()
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
						Entity: func(entity ecs.ID) bool { return entity != attacker },
					})

				if rayHit.Entity != ecs.NullID {
					transform := T_attack.Inv().Mul(world.GetGlobalTransform(rayHit.Entity))

					io.EnqueueEntityUpdate(weapon,
						func(_ *UpdateParams, weapon Entity2, io IO) {
							state := weapon.ScriptState().(WeaponPhysgun)
							defer func() { weapon.SetScriptState(state) }()

							state.HeldEntity = rayHit.Entity
							state.Transform = transform
						})
				}

			case holdingEntity && triggerHeld:
				// TODO: this doesn't work correctly when we're touching an object
				// that's parented to something.
				transform := T_attack.Mul(physgun.Transform)

				io.EnqueueEntityUpdate(physgun.HeldEntity,
					func(_ *UpdateParams, id Entity2, io IO) {
						id.SetTransform(transform.TRS())
						id.SetVelocity(Velocity{})
					})

			case holdingEntity && !triggerHeld:
				io.EnqueueEntityUpdate(weapon,
					func(_ *UpdateParams, weapon Entity2, io IO) {
						state := weapon.ScriptState().(WeaponPhysgun)
						defer func() { weapon.SetScriptState(state) }()

						state.HeldEntity = ecs.NullID
					})
			}

			return Recoil{}
		},
	}
}
