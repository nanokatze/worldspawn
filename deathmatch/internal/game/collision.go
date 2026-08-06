package game

import (
	"fmt"
	"iter"
	"math"
	"unique"

	"worldspawn/internal/ecs"
	"worldspawn/internal/physics"
)

type CollisionLayer uint8

const (
	CollisionLayerNoCollision = iota
	CollisionLayerBackground
	CollisionLayerProp
	CollisionLayerProjectile
	CollisionLayerMovingKinematic // TODO: once we decouple motion types, kill this
	// TODO: I think we need another layer for triggers :thinking:
	numCollisionLayers
)

var collisionLayerRules = func() physics.LayerCollisionRules {
	const F = false
	const T = true

	// TODO: I still hate this, I think I'd prefer a setter api :/
	return physics.LayerCollisionRules{
		/*               No No Mo Pr    */
		/*               Co nM vi oj    */
		/*               ll ov ng ec    */
		/*               is in    ti    */
		/*               io  g    le    */
		/*                n        s    */
		/* NoCollision */ F, F, F, F, F,
		/* NonMoving      */ F, T, T, F,
		/* Moving            */ T, T, T,
		/* Projectiles          */ F, T,
		/* MovingKinematic         */ T,
	}
}()

// TODO: decouple this back again. We'll make CollisionLayerNoCollision bodies
// legal, but we'll be able to exclude CollisionLayerNoCollision && non-Dynamic
// bodies entirely from the physics system.
var collisionLayerMotionType = [numCollisionLayers]int{
	CollisionLayerNoCollision:     0,
	CollisionLayerBackground:      0,
	CollisionLayerProp:            2,
	CollisionLayerProjectile:      2,
	CollisionLayerMovingKinematic: 1,
}

// TODO: generate this plx
var collisionLayerFromString = map[string]CollisionLayer{
	"NoCollision":     CollisionLayerNoCollision,
	"NonMoving":       CollisionLayerBackground,
	"Moving":          CollisionLayerProp,
	"Projectiles":     CollisionLayerProjectile,
	"MovingKinematic": CollisionLayerMovingKinematic,
}

func (collisionLayer *CollisionLayer) UnmarshalText(text []byte) error {
	tmp, ok := collisionLayerFromString[string(text)]
	if !ok {
		return fmt.Errorf("unknown collision layer %v", string(text))
	}
	*collisionLayer = tmp
	return nil
}

// TODO: hide this away, next to where physics is initialized.
const (
	broadPhaseLayerMoving = iota
	broadPhaseLayerNonMoving
)

var collisionLayerToBroadPhaseLayer = [numCollisionLayers]uint8{
	CollisionLayerBackground: broadPhaseLayerNonMoving,
}

func (e Entity) SetCollisionLayer(v CollisionLayer) { e.world.CollisionLayer.Store(e.id.Index(), v) }

func (e Entity) SetCollisionGeometry(v unique.Handle[string]) {
	e.world.CollisionGeometry.Store(e.id.Index(), v)
}

type RayHit struct {
	Entity Entity
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
		result.Entity = Entity{world, world.Table.IDs().Index(int(collector.closestHit.BodyID))}
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
