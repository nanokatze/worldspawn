package game

import (
	"math"

	"worldspawn/internal/ecs"
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
// (https://github.com/nanokatze/worldspawn-deathmatch/issues/12)
// TODO: we should have a separate Explosion struct which will describe some things that Impact does (but not all)
func (world *World) explosion(
	stx ScriptContext,
	impact Impact, // TODO: flatten/inline the relevant fields instead of passing Impact
	T gmath.Affine3f64, // TODO: move this to be the first parameter?
	df distributionFunction,
	radius float32,
	resolution float32,
	queryFilters QueryFilters,
) {
	spat := resolution / (4 * math.Pi)
	nrays := math.Ceil(1.0 / float64(spat))
	// Round things up
	spat = float32(1.0 / float64(nrays))

	// TODO: assert nrays >= 6?

	// TODO: outline tracing into its own function

	type result struct {
		dvel Velocity
		dmg  float32
	}

	results := make(map[ecs.ID]result)

	for u := range fibonacciLattice(int64(nrays)) {
		d, pdf := df(u)

		d = T.M.Mulv(d)

		rayHit := world.TraceRay(
			physics.Ray{
				Origin:    T.T,
				Direction: d.Normalize(),
				TMax:      radius,
			},
			queryFilters)
		if !rayHit.Entity.IsValid() {
			continue
		}

		dmg := pdf * spat

		id := rayHit.Entity.ID()

		tmp := results[id]
		tmp.dvel = tmp.dvel.Add(Velocity{Linear: d}.Scale(dmg))
		tmp.dmg += dmg
		results[id] = tmp
	}

	for entityID, result := range results {
		impact := Impact{
			Attacker: impact.Attacker,
			Type:     impact.Type,
			Δv:       result.dvel.Scale(impactForceFactor[impact.Type] * float32(impact.Damage)),
			Damage:   result.dmg * impact.Damage,
		}

		// TODO: we could skip validation here.
		stx.Update(world.Entity(entityID), impact.Apply)
	}
}

func sphericalExplosion(u gmath.Vec2f32) (gmath.Vec3f32, float32) { return sampleSphere(u), 1 }
