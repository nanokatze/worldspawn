package game

import (
	"iter"
	"math"

	"worldspawn/internal/ecs"
	"worldspawn/internal/physics"
)

type RayHit struct {
	Entity Entity2
	T      float32
}

// TODO: make this opaque and do functional config instead
type QueryFilters struct {
	// Should use Entity2
	Entity func(ecs.ID) bool
}

// TODO: have it return (RayHit, bool) actually?
func (world *World) TraceRay(ray physics.Ray, filters QueryFilters) RayHit {
	var collector closestHitCollector
	collector.world = world
	collector.filters = filters
	collector.closestHit = physics.SceneRayHit{
		BodyID: 0xffffffff,
		Geometry: physics.RayHit{
			T: float32(math.Inf(1)),
		},
	}
	world.physics.TraceRay(ray, &collector)

	var result RayHit
	if collector.closestHit.BodyID != 0xffffffff {
		result.Entity = Entity2{world, world.Table.IDs().Index(int(collector.closestHit.BodyID))}
	}
	return result
}

// The user can just break the loop after the first hit to achieve "terminate on first hit"
func (world *World) TraceRayAllHits(ray physics.Ray) iter.Seq[RayHit] {
	panic("not implemented")
}

type closestHitCollector struct {
	world   *World
	filters QueryFilters

	closestHit physics.SceneRayHit
}

func (collector *closestHitCollector) FilterBody(body physics.BodyID) bool {
	if collector.filters.Entity == nil {
		return true
	}
	return collector.filters.Entity(collector.world.Table.IDs().Index(int(body)))
}

func (collector *closestHitCollector) FilterLayer(layer int) bool {
	return true
}

func (collector *closestHitCollector) Hit(hit physics.SceneRayHit) physics.QueryPipelineControl {
	collector.closestHit = hit
	return physics.AcceptHit
}
