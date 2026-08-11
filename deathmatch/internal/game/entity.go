package game

import (
	"fmt"
	"reflect"
	"unique"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
	"worldspawn/internal/loaders/skeleton"
	"worldspawn/internal/physics"
)

// Entity represents a reference to an entity
type Entity struct {
	world *Columns // TODO: change this to columns
	id    ecs.ID
}

func (e Entity) String() string { return fmt.Sprintf("%d", e.id) }

func (e Entity) IsValid() bool { return e.id != 0 }

func (e Entity) mustBeValid() {
	// TODO: we can check e.world instead I guess
	if !e.IsValid() {
		panic("must be valid")
	}
}

func (e Entity) ID() ecs.ID {
	e.mustBeValid()
	return e.id
}

// TODO: rename to something nicer
func (e Entity) Clear() {
	if _, ok := e.world.physicsBodyExists.Get(e.id); ok {
		// We have to do this because ClearRow unsets the bit in
		// physicsBodyExists, so we end up with an orphan physics body.
		e.world.physics.RemoveBody(physics.BodyID(e.id.Index()))
	}
	e.world.Table.ClearRow(e.id)
}

// TODO: autogenerate most of these

// TODO: remove this, the users should look up the script associated with a
// particular script state by themselves
func (e Entity) Script() script {
	return Scripts[reflect.TypeOf(e.world.ScriptState.Load(e.id.Index()))]
}

func (e Entity) ScriptState() ScriptState {
	return e.world.ScriptState.Load(e.id.Index())
}

func (e Entity) SetScriptState(v ScriptState) {
	e.world.ScriptState.Store(e.id.Index(), v)
}

func (e Entity) SetNextThink(v Time) {
	e.world.NextThink.Store(e.id.Index(), v)
}

func (e Entity) Parent() ecs.ID {
	return e.world.Parent.Load(e.id.Index())
}

func (e Entity) SetParent(v ecs.ID) {
	e.world.SetParent(e.id, v)
}

func (e Entity) ParentBone() unique.Handle[string] {
	return e.world.ParentBone.Load(e.id.Index())
}

func (e Entity) SetParentBone(v unique.Handle[string]) {
	e.world.ParentBone.Store(e.id.Index(), v)
}

func (e Entity) Transform() gmath.TRS3f64 {

	tr := e.world.TransformTR.Load(e.id.Index())
	s := e.world.TransformS.Load(e.id.Index())
	return gmath.TRS3f64{tr.T, tr.R, s}
}

func (e Entity) SetTransform(v gmath.TRS3f64) {

	// TODO: validate that the transform is invertible? We might wanna ban non-invertible transforms

	e.world.TransformTR.Store(e.id.Index(), TR3f64{v.T, v.R})
	e.world.TransformS.Store(e.id.Index(), v.S)
}

func (e Entity) SetTransformTR(v TR3f64) {
	e.world.TransformTR.Store(e.id.Index(), v)
}

func (e Entity) Skeleton() unique.Handle[string] {
	return e.world.Skeleton.Load(e.id.Index())
}

func (e Entity) SetSkeleton(v unique.Handle[string]) {
	e.world.Skeleton.Store(e.id.Index(), v)
}

func (e Entity) Pose() skeleton.Pose {
	return e.world.pose[e.id.Index()]
}

// Note that pose is not replicated
//
// TODO: change up the api to encourage slice reuse
func (e Entity) SetPose(v skeleton.Pose) {
	e.world.pose[e.id.Index()] = v
}

func (e Entity) SetShouldSetOffFuseOnImpact(v bool) {
	// TODO: raaah just have it be a plain bitset already!
	if v {
		e.world.ShouldSetOffFuseOnImpact.Store(e.id.Index(), struct{}{})
	} else {
		e.world.ShouldSetOffFuseOnImpact.Delete(e.id)
	}
}

func (e Entity) SetCollisionLayer(v CollisionLayer) {
	e.world.CollisionLayer.Store(e.id.Index(), v)
}

func (e Entity) SetCollisionGeometry(v unique.Handle[string]) {
	e.world.CollisionGeometry.Store(e.id.Index(), v)
}

func (e Entity) Velocity() Velocity {
	return e.world.Velocity.Load(e.id.Index())
}

func (e Entity) SetVelocity(v Velocity) {
	e.world.Velocity.Store(e.id.Index(), v)
}

func (e Entity) SetPhysicsMassOverride(v float32) {
	e.world.PhysicsMassOverride.Store(e.id.Index(), v)
}

func (e Entity) SetVisibilityCondition(v VisibilityCondition) {
	e.world.VisibilityCondition.Store(e.id.Index(), v)
}

func (e Entity) SetCosmeticOffset(v CosmeticOffset) {
	e.world.CosmeticOffset.Store(e.id.Index(), v)
}

func (e Entity) SetRenderingGeometry(v unique.Handle[string]) {
	e.world.RenderingGeometry.Store(e.id.Index(), v)
}

func (e Entity) SetSoundEffect(v SoundEmitter) {
	e.world.SoundEffect.Store(e.id.Index(), v)
}

func (e Entity) MarkForDeletion() {
	e.world.delete.Store(e.id.Index(), true)
}
