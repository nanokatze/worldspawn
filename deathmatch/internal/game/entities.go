package game

import (
	"fmt"
	"reflect"
	"unique"

	"worldspawn/internal/ecs"
	"worldspawn/internal/ecs/bitslice"
	"worldspawn/internal/gmath"
	"worldspawn/internal/loaders/skeleton"
	"worldspawn/internal/physics"
)

type EntityID = ecs.ID

// TODO: fold almost all scripty stuff into the same column? Although I guess
// that's more up to how we wanna represent things for authoring etc. I'm not
// sure it would make sense to fold animation graph script into the same column
// as game logic script. However...
// TODO: make this private once the API for replication is functioning
type Entities struct {
	// This is manually replicated for now.
	// TODO: replace with a simple array of generations
	Table *ecs.Table `worldspawn:"dontreplicate"`

	// Entity programmability

	Name ecs.Column[unique.Handle[string]]

	ScriptState ecs.Column[ScriptState]

	// TODO: this will need to be a wheel rather than a plain column
	NextThink ecs.Column[Time]

	// TODO: require that entities that we're do collision/physics for have no
	// parent. Or maybe allow that somehow, e.g. by having children be joined
	// with the collision geometry of the parent?

	// Transform

	// Do not access this column directly; use {Get,Set}Parent. Specifies the
	// parent entity.
	Parent ecs.Column[EntityID]
	// The bone in the parent's skeleton that transforms this entity.
	ParentBone ecs.Column[unique.Handle[string]]
	// Do not access this column directly; use {Get,Set}Transform or
	// GetGlobalTransform instead.
	//
	// The translation and rotation parts of the entity's parent-relative
	// transform.
	TransformTR ecs.Column[TR3f64]
	// Do not access this column directly; use {Get,Set}Transform and
	// GetGlobalTransform instead.
	//
	// The scale and shearing part of the entity's parent-relative transform.
	//
	// It is possible for an entity to have a TransformTR but not TransformS, in
	// which case no scaling or shearing is applied to the entity.
	// TODO: we'll later hard-require TransformS to always be set to something valid.
	TransformS ecs.Column[gmath.Mat3x3Uf32]

	// TODO: make skeletons part of geometry? Ok actually wait which geometry
	// lol. We'll need to think more about this.
	Skeleton ecs.Column[unique.Handle[string]]

	pose []skeleton.Pose `worldspawn:"dontreplicate"`

	// Collision

	CollisionLayer    ecs.Column[CollisionLayer]
	CollisionGeometry ecs.Column[unique.Handle[string]]

	CollisionSensor ecs.Column[struct{}]

	// Motion

	// MotionType       ecs.Column[MotionType]
	// MotionProperties ecs.Column[MotionProperties]

	Velocity ecs.Column[Screw3]

	// TODO: remove GravityFactor, Physics{Mass,Inertia}Override in favor of the
	// MotionProperties column. Or better yet, CollisionGeometry script setting
	// that.

	GravityFactor          ecs.Column[float32]
	PhysicsMassOverride    ecs.Column[float32]
	PhysicsInertiaOverride ecs.Column[gmath.Mat4x4f32]

	// TODO: kill this column and handle it at prefab instantination
	CollectionInstance ecs.Column[CollectionInstance] `worldspawn:"dontreplicate"`

	// Other gameplay stuff

	// Whether the fuse of certain projectiles should be set off when they
	// collide with this entity.
	ShouldSetOffFuseOnImpact ecs.Column[struct{}]

	// Renderer

	VisibilityCondition ecs.Column[VisibilityCondition]

	CosmeticOffset ecs.Column[CosmeticOffset]

	RenderingGeometry ecs.Column[unique.Handle[string]]

	// TODO: so the way we'll probably go about sounds is by having 2 networked
	// columns and one shadow column. The shadow column will feed the actual
	// audio player, so it should contain basically sound files and t0 (a mix of
	// game time and sample count) and the networked columns be the "audio
	// program" identifier and audio program state. For now we can start with
	// just the shadow column.

	// TODO: rethink sounds
	SoundEffect      ecs.Column[SoundEmitter]
	SoundEffectState ecs.Column[LoopedSound] // TODO: kill this column

	// Deadlines of speculatively spawned entities.
	//
	// TODO: rename
	Speculation ecs.Column[Time] `worldspawn:"dontreplicate"`

	// TODO: wrap these two into a type that implements Column interface
	physics           *physics.System      `worldspawn:"dontreplicate"`
	physicsBodyExists ecs.Column[struct{}] `worldspawn:"dontreplicate"`

	// Entities marked for deletion
	delete bitslice.BitSlice `worldspawn:"dontreplicate"`
}

// Entity represents a reference to an entity
type Entity struct {
	_        [0]func()
	entities *Entities
	id       EntityID
}

func (e Entity) String() string { return fmt.Sprintf("%d", e.id) }

func (e Entity) IsValid() bool { return e.id != 0 }

func (e Entity) ID() EntityID {
	if !e.IsValid() {
		return ecs.NullID
	}
	return e.id
}

// TODO: rename to something nicer
func (e Entity) Clear() {
	if _, ok := e.entities.physicsBodyExists.Get(e.id); ok {
		// We have to do this because ClearRow unsets the bit in
		// physicsBodyExists, so we end up with an orphan physics body.
		e.entities.physics.RemoveBody(physics.BodyID(e.id.Index()))
	}
	e.entities.Table.ClearRow(e.id)
}

// TODO: autogenerate most of these

// TODO: remove this, the users should look up the script associated with a
// particular script state by themselves
func (e Entity) Script() script {
	return Scripts[reflect.TypeOf(e.entities.ScriptState.Load(e.id.Index()))]
}

func (e Entity) ScriptState() ScriptState {
	return e.entities.ScriptState.Load(e.id.Index())
}

func (e Entity) SetScriptState(v ScriptState) {
	e.entities.ScriptState.Store(e.id.Index(), v)
}

func (e Entity) SetNextThink(v Time) {
	e.entities.NextThink.Store(e.id.Index(), v)
}

func (e Entity) Parent() EntityID {
	return e.entities.Parent.Load(e.id.Index())
}

func (e Entity) SetParent(v EntityID) {
	e.entities.SetParent(e.id, v)
}

func (e Entity) ParentBone() unique.Handle[string] {
	return e.entities.ParentBone.Load(e.id.Index())
}

func (e Entity) SetParentBone(v unique.Handle[string]) {
	e.entities.ParentBone.Store(e.id.Index(), v)
}

func (e Entity) Transform() gmath.Affine3TRSf64 {
	tr := e.entities.TransformTR.Load(e.id.Index())
	s := e.entities.TransformS.Load(e.id.Index())
	return gmath.Affine3TRSf64{tr.T, tr.R, s}
}

func (e Entity) SetTransform(v gmath.Affine3TRSf64) {
	// TODO: validate that the transform is invertible? We might wanna ban non-invertible transforms

	e.entities.TransformTR.Store(e.id.Index(), TR3f64{v.T, v.R})
	e.entities.TransformS.Store(e.id.Index(), v.S)
}

func (e Entity) SetTransformTR(v TR3f64) {
	e.entities.TransformTR.Store(e.id.Index(), v)
}

func (e Entity) Skeleton() unique.Handle[string] {
	return e.entities.Skeleton.Load(e.id.Index())
}

func (e Entity) SetSkeleton(v unique.Handle[string]) {
	e.entities.Skeleton.Store(e.id.Index(), v)
}

func (e Entity) Pose() skeleton.Pose {
	return e.entities.pose[e.id.Index()]
}

// Note that pose is not replicated
//
// TODO: change up the api to encourage slice reuse
func (e Entity) SetPose(v skeleton.Pose) {
	e.entities.pose[e.id.Index()] = v
}

func (e Entity) SetShouldSetOffFuseOnImpact(v bool) {
	// TODO: raaah just have it be a plain bitset already!
	if v {
		e.entities.ShouldSetOffFuseOnImpact.Store(e.id.Index(), struct{}{})
	} else {
		e.entities.ShouldSetOffFuseOnImpact.Delete(e.id)
	}
}

func (e Entity) SetCollisionLayer(v CollisionLayer) {
	e.entities.CollisionLayer.Store(e.id.Index(), v)
}

func (e Entity) SetCollisionGeometry(v unique.Handle[string]) {
	e.entities.CollisionGeometry.Store(e.id.Index(), v)
}

func (e Entity) Velocity() Screw3 {
	return e.entities.Velocity.Load(e.id.Index())
}

func (e Entity) SetVelocity(V Screw3) {
	e.entities.Velocity.Store(e.id.Index(), V)
}

func (e Entity) SetPhysicsMassOverride(v float32) {
	e.entities.PhysicsMassOverride.Store(e.id.Index(), v)
}

func (e Entity) SetVisibilityCondition(v VisibilityCondition) {
	e.entities.VisibilityCondition.Store(e.id.Index(), v)
}

func (e Entity) SetCosmeticOffset(v CosmeticOffset) {
	e.entities.CosmeticOffset.Store(e.id.Index(), v)
}

func (e Entity) SetRenderingGeometry(v unique.Handle[string]) {
	e.entities.RenderingGeometry.Store(e.id.Index(), v)
}

func (e Entity) SetSoundEffect(v SoundEmitter) {
	e.entities.SoundEffect.Store(e.id.Index(), v)
}

func (e Entity) MarkForDeletion() {
	e.entities.delete.Store(e.id.Index(), true)
}
