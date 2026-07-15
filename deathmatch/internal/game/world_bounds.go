package game

import (
	"math"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

// TODO: make this a method on VecN?
func lessthan(a, b gmath.Vec3f64) bool {
	return a[0] < b[0] && a[1] < b[1] && a[2] < b[2]
}

// TODO: we'll probably want to use physics engine in some way with infinite
// plane shapes.
func (world *World) handleOutOfBoundsEntities(info *UpdateParams) {
	// TODO: specify the bounds on SceneGlobals or something
	bounds := [2]gmath.Vec3f64{
		{
			math.Inf(-1),
			math.Inf(-1),
			-100,
		},
		{
			math.Inf(1),
			math.Inf(1),
			math.Inf(1),
		},
	}

	// TODO: I don't actually suppose this matters for things that don't run
	// on physics?
	for id, tr := range ecs.All(&world.TransformTR) {
		if world.GetParent(id) != ecs.NullID {
			// TODO: figure out a good way to handle entities with parents
			continue
		}

		if lessthan(bounds[0], tr.T) && lessthan(tr.T, bounds[1]) {
			// Contained within the bounds
			continue
		}

		world.logger.Warn("entity out of bounds", "id", id, "T", tr.T)

		// TODO: check if the entity's script implements OutOfBounds and
		// poke that instead of this thing.

		world.Entities.delete.Store(id.Index(), true)
	}

	world.deleteMarkedEntities()
}
