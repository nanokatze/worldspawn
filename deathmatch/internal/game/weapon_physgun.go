package game

import (
	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
	"worldspawn/internal/physics"
)

type WeaponPhysgun struct {
	Holding   bool
	Object    ecs.ID
	Transform gmath.Affine3f64
}

func (WeaponPhysgun) entity() {}

var _ Entity = WeaponPhysgun{}
var _ Weapon = WeaponPhysgun{}

func (physgun WeaponPhysgun) CreateProp(scene *Scene, info *UpdateParams) ecs.ID {
	return scene.CreateEntity(info)
}

func (physgun WeaponPhysgun) WeaponSubstep(
	scene *Scene,
	weapon ecs.ID,
	propIDs []ecs.ID,
	shooter ecs.ID,
	T gmath.Affine3f64,
	v Velocity,
	buttons WeaponButtons,
	info *UpdateParams,
) Recoil {
	switch {
	case !physgun.Holding && buttons&WeaponTrigger != 0:
		var collector physgunRayQueryPipeline
		collector.shooter = physics.BodyID(shooter.Index())
		collector.hit.BodyID = 0xffffffff

		scene.physicsSystem.TraceRay(
			physics.Ray{
				Origin:    T.T,
				Direction: T.M.Mulv(forward).Normalize(),
				TMax:      1000,
			},
			&collector)

		if collector.hit.BodyID != 0xffffffff {
			physgun.Holding = true
			physgun.Object = scene.Table.IDs().Index(int(collector.hit.BodyID))
			physgun.Transform = T.Inv().Mul(scene.GetGlobalTransform(physgun.Object))
		}

		scene.Entity.Set(weapon, physgun)

	case physgun.Holding && buttons&WeaponTrigger == 0:
		physgun.Holding = false

		scene.Entity.Set(weapon, physgun)

	case physgun.Holding:
		// TODO: this doesn't work correctly when we're touching an object
		// that's parented to something. We'll want a SetGlobalTransform but
		// leave a note that it's intended to be debugging only or
		// something.
		transform := T.Mul(physgun.Transform)

		scene.SendMessage(physgun.Object,
			func(scene *Scene, id ecs.ID, updateParams *UpdateParams) {
				scene.SetTransform(id, transform.TRS())
				scene.Velocity.Set(id, Velocity{})
			})
	}

	return Recoil{}
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
