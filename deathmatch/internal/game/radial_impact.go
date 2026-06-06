package game

import (
	"iter"
	"math"

	"worldspawn/internal/gmath"
	"worldspawn/internal/physics"
)

// TODO: this should be an interface
// TODO: prefix with something? (e.g. explosion)
type distributionFunction func(gmath.Vec2f32) (gmath.Vec3f32, float32)

// TODO: allow the user to specify filters
// TODO: allow for a little bit of extra linear distance falloff to model dissipation?
func (scene *Scene) radialImpact(
	impact Impact,
	T gmath.Affine3f64, // TODO: move this to be the first parameter?
	df distributionFunction,
	radius float32,
	resolution float32, // TODO: reformulate it in terms of something that doesn't require adjustment along with the radius
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

		scene.physicsSystem.TraceRay(
			physics.Ray{
				Origin:    T.T,
				Direction: d.Normalize(),
				TMax:      radius,
			},
			&collector)
		if collector.closestHit.BodyID == 0xffffffff {
			continue
		}

		dmg := impact.Damage * pdf * float32(spat)

		tmp := results[collector.closestHit.BodyID]
		tmp.dvel = tmp.dvel.Add(Velocity{Linear: d.Scale(dmg * impactTypes[impact.Type])})
		tmp.dmg += dmg
		results[collector.closestHit.BodyID] = tmp
	}

	for bodyID, result := range results {
		entityID := scene.Table.IDs().Index(int(bodyID))

		impact := Impact{
			Type:      impact.Type,
			Δv:        result.dvel,
			Damage:    result.dmg,
			Inflictor: impact.Inflictor,
		}

		scene.SendMessage(entityID, impact.Apply)
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

// TODO: move these somewhere else

func sampleSphere(u gmath.Vec2f32) gmath.Vec3f32 {
	theta := float64(2 * math.Pi * u[0])
	phi := math.Acos(float64(1 - 2*u[1]))
	sinTheta, cosTheta := math.Sincos(theta)
	sinPhi, cosPhi := math.Sincos(phi)
	return gmath.Vec3Convert[float32](gmath.Vec3f64{
		cosTheta * sinPhi,
		sinTheta * sinPhi,
		cosPhi,
	})
}

// TODO: https://extremelearning.com.au/evenly-distributing-points-on-a-sphere/
func fibonacciLattice(n int64) iter.Seq[gmath.Vec2f32] {
	goldenRatio := (1 + math.Sqrt(5)) / 2

	return func(yield func(gmath.Vec2f32) bool) {
		for i := range n {
			p := gmath.Vec2f64{math.Mod(float64(i)/goldenRatio, 1), float64(i) / float64(n-1)}
			if !yield(gmath.Vec2Convert[float32](p)) {
				break
			}
		}
	}
}
