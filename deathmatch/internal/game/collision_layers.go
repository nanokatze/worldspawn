package game

import "worldspawn/internal/physics"

type CollisionLayer uint8

// TODO: change numbering to be from 0 in preparation to switching to a plain
// slice (with fast iter over non-zeros) for CollisionLayer
const (
	CollisionLayerNonMoving CollisionLayer = iota
	CollisionLayerMoving
	CollisionLayerProjectiles
	CollisionLayerMovingKinematic // used by character controllers; TODO: rename
	// TODO: I think we need another layer for triggers :thinking:
	numCollisionLayers
)

var collisionLayerRules = func() physics.LayerCollisionRules {
	const F = false
	const T = true

	// TODO: I still hate this, I think I'd prefer a setter api :/
	return physics.LayerCollisionRules{
		/*                   No Mo Pr    */
		/*                   nM vi oj    */
		/*                   ov ng ec    */
		/*                   in    ti    */
		/*                   g     le    */
		/*                         s     */
		/* NonMoving       */ F, T, T, F,
		/* Moving             */ T, T, T,
		/* Projectiles           */ F, T,
		/* MovingKinematic          */ T,
	}
}()

var collisionLayerMotionType = map[CollisionLayer]int{
	CollisionLayerNonMoving:       0,
	CollisionLayerMoving:          2,
	CollisionLayerProjectiles:     2,
	CollisionLayerMovingKinematic: 1,
}

// TODO: generate this plx
var collisionLayerFromString = map[string]CollisionLayer{
	"NonMoving":       CollisionLayerNonMoving,
	"Moving":          CollisionLayerMoving,
	"Projectiles":     CollisionLayerProjectiles,
	"MovingKinematic": CollisionLayerMovingKinematic,
}

/*
	func (physicsLayer *PhysicsLayer) UnmarshalText(text []byte) error {
		tmp, ok := physicsLayerFromString[string(text)]
		if !ok {
			return errors.New("unknown shape type")
		}
		*physicsLayer = tmp
		return nil
	}
*/

const (
	broadPhaseLayerNonMoving = iota
	broadPhaseLayerMoving
)

var collisionLayerToBroadPhaseLayer = [numCollisionLayers]uint8{
	CollisionLayerNonMoving:       broadPhaseLayerNonMoving,
	CollisionLayerMoving:          broadPhaseLayerMoving,
	CollisionLayerProjectiles:     broadPhaseLayerMoving,
	CollisionLayerMovingKinematic: broadPhaseLayerMoving,
}
