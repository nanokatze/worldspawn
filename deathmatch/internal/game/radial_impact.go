package game

import (
	"math"

	"worldspawn/internal/gmath"
	"worldspawn/internal/physics"
)

// TODO: this should be an interface
// TODO: prefix with something? (e.g. explosion)
type distributionFunction func(gmath.Vec2f32) (gmath.Vec3f32, float32)

// TODO: allow the user to specify filters
// TODO: allow for a little bit of extra linear distance falloff to model dissipation?
// TODO: reformulate resolution in terms of something that doesn't require
// adjustment along with the radius
// (https://github.com/nanokatze/worldspawn/issues/152)
func (world *World) radialImpact(
	impact Impact,
	T gmath.Affine3f64, // TODO: move this to be the first parameter?
	df distributionFunction,
	radius float32,
	resolution float32,
) {
	spat := resolution / (4 * math.Pi)
	nrays := math.Ceil(1.0 / float64(spat))
	// Round things up
	spat = float32(1.0 / float64(nrays))

	// TODO: assert nrays >= 6?

	// TODO: outline tracing into its own function

	var collector explosionHitCollector

	type result struct {
		dvel Velocity
		dmg  float32
	}

	results := make(map[physics.BodyID]result)

	for u := range fibonacciLattice(int64(nrays)) {
		d, pdf := df(u)

		d = T.M.Mulv(d)

		collector.closestHit = physics.SceneRayHit{
			BodyID: 0xffffffff,
			Geometry: physics.RayHit{
				T: float32(math.Inf(1)),
			},
		}

		world.physics.TraceRay(
			physics.Ray{
				Origin:    T.T,
				Direction: d.Normalize(),
				TMax:      radius,
			},
			&collector)
		if collector.closestHit.BodyID == 0xffffffff {
			continue
		}

		dmg := pdf * spat

		tmp := results[collector.closestHit.BodyID]
		tmp.dvel = tmp.dvel.Add(Velocity{Linear: d.Scale(dmg * impactForceFactor[impact.Type])})
		tmp.dmg += dmg
		results[collector.closestHit.BodyID] = tmp
	}

	for bodyID, result := range results {
		entityID := world.Table.IDs().Index(int(bodyID))

		result.dvel.Linear = result.dvel.Linear.Scale(float32(impact.Damage))

		impact := Impact{
			Type:      impact.Type,
			Δv:        result.dvel,
			Damage:    int32(result.dmg * float32(impact.Damage)),
			Inflictor: impact.Inflictor,
		}

		world.EnqueueEntityUpdate(entityID, impact.Apply)
	}
}

type explosionHitCollector struct {
	closestHit physics.SceneRayHit
}

func (collector *explosionHitCollector) FilterLayer(layer int) bool {
	// TODO: we really should do a bitmap tbh
	return layer == int(CollisionLayerNonMoving) ||
		layer == int(CollisionLayerMoving) ||
		layer == int(CollisionLayerMovingKinematic)
}

func (collector *explosionHitCollector) Hit(hit physics.SceneRayHit) physics.QueryPipelineControl {
	collector.closestHit = hit
	return physics.AcceptHit
}

func sphericalExplosion(u gmath.Vec2f32) (gmath.Vec3f32, float32) { return sampleSphere(u), 1 }
