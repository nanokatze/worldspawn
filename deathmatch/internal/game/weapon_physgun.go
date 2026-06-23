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
			physgun, _ := world.GetEntity[WeaponPhysgun](weapon)

			holdingEntity := world.EntityExists(physgun.HeldEntity)
			triggerHeld := buttons&WeaponTrigger != 0

			switch {
			case !holdingEntity && triggerHeld:
				var collector physgunRayQueryPipeline
				collector.shooter = physics.BodyID(attacker.Index())
				collector.hit.BodyID = 0xffffffff

				world.physics.TraceRay(
					physics.Ray{
						Origin:    T_attack.T,
						Direction: T_attack.M.Mulv(forward).Normalize(),
						TMax:      1000,
					},
					&collector)

				if collector.hit.BodyID != 0xffffffff {
					world.EnqueueEntityUpdate(weapon,
						func(world *World, id ecs.ID, _ *UpdateParams) {
							physgun, _ := world.GetEntity[WeaponPhysgun](weapon)
							// TODO: precompute all of these
							physgun.HeldEntity = world.Table.IDs().Index(int(collector.hit.BodyID))
							physgun.Transform = T_attack.Inv().Mul(world.GetGlobalTransform(physgun.HeldEntity))
							world.Entity.Set(weapon, physgun)
						})
				}

			case holdingEntity && triggerHeld:
				// TODO: this doesn't work correctly when we're touching an object
				// that's parented to something.
				transform := T_attack.Mul(physgun.Transform)

				world.EnqueueEntityUpdate(physgun.HeldEntity,
					func(world *World, id ecs.ID, _ *UpdateParams) {
						world.SetTransform(id, transform.TRS())
						world.Velocity.Set(id, Velocity{})
					})

			case holdingEntity && !triggerHeld:
				world.EnqueueEntityUpdate(weapon,
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

type physgunRayQueryPipeline struct {
	shooter physics.BodyID
	hit     physics.SceneRayHit
}

func (pipeline *physgunRayQueryPipeline) FilterBody(body physics.BodyID) bool {
	return body != pipeline.shooter
}

func (pipeline *physgunRayQueryPipeline) Hit(hit physics.SceneRayHit) physics.QueryPipelineControl {
	pipeline.hit = hit
	return physics.AcceptHit
}
