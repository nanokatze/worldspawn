package game

import (
	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
	"worldspawn/internal/physics"
)

type WeaponPhysgun struct {
	HeldEntity ecs.ID
	Transform  gmath.Affine3f64
}

func (WeaponPhysgun) entity() {}

var _ Entity = WeaponPhysgun{}

func init() {
	scripts["weapon_physgun"] = script{
		WeaponThink: func(
			world *World,
			weapon ecs.ID,
			props []ecs.ID,
			attacker ecs.ID,
			T_attack gmath.Affine3f64,
			v_attack Velocity,
			buttons WeaponButtons,
			info *UpdateParams) Recoil {
			io := IO{world, weapon}

			physgun, _ := world.GetEntity[WeaponPhysgun](weapon)

			holdingEntity := world.EntityExists(physgun.HeldEntity)
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
					io.EnqueueEntityUpdate(weapon,
						func(world *World, id ecs.ID, _ *UpdateParams) {
							physgun, _ := world.GetEntity[WeaponPhysgun](weapon)
							// TODO: precompute all of these
							physgun.HeldEntity = rayHit.Entity
							physgun.Transform = T_attack.Inv().Mul(world.GetGlobalTransform(physgun.HeldEntity))
							world.Entity.Set(weapon, physgun)
						})
				}

			case holdingEntity && triggerHeld:
				// TODO: this doesn't work correctly when we're touching an object
				// that's parented to something.
				transform := T_attack.Mul(physgun.Transform)

				io.EnqueueEntityUpdate(physgun.HeldEntity,
					func(world *World, id ecs.ID, _ *UpdateParams) {
						world.SetTransform(id, transform.TRS())
						world.Velocity.Set(id, Velocity{})
					})

			case holdingEntity && !triggerHeld:
				io.EnqueueEntityUpdate(weapon,
					func(world *World, weapon ecs.ID, _ *UpdateParams) {
						physgun, _ := world.GetEntity[WeaponPhysgun](weapon)
						physgun.HeldEntity = ecs.NullID
						world.Entity.Set(weapon, physgun)
					})
			}

			return Recoil{}
		},
	}
}
