package physics

// #include "c/physics.h"
import "C"

// xasdasd

import (
	"errors"
	"unsafe"

	"worldspawn/internal/geometry"
)

// TODO: redo this package so that it's relatively straightforward bindings to
// JPH. Callbacks should use C convention by default

// TODO: we should move most of worldspawn/physics.go code into this package. Or
// not. IDK. We'll need to think hard about it.
//
// TODO: see if we can move .cpp files and .go in the same directory.

type BodyID uint32

type System C.Physics

type Shape C.Shape

// Used by deserializer only
type ShapeKind int

const (
	_ ShapeKind = iota
	ShapeSphere
	ShapeBox
	ShapeCylinder
	ShapeConvexHull
	ShapeMesh
)

var shapeKindFromString = map[string]ShapeKind{
	"Sphere":     ShapeSphere,
	"Box":        ShapeBox,
	"Cylinder":   ShapeCylinder,
	"ConvexHull": ShapeConvexHull,
	"Mesh":       ShapeMesh,
}

func (shapeType *ShapeKind) UnmarshalText(text []byte) error {
	var ok bool
	if *shapeType, ok = shapeKindFromString[string(text)]; !ok {
		return errors.New("unknown shape type")
	}
	return nil
}

type Stuff struct {
	Material       int
	VertexBuffer   int
	VertexCount    int
	TriangleBuffer int
	TriangleCount  int
}

type Triangle struct {
	VertexIndices [3]uint32
	MaterialIndex uint32
}

type BroadPhaseLayer uint8

func NewSystem(
	BroadPhaseLayerCount int,
	ObjectLayerCount int,
	ObjectLayerToBroadPhaseLayer []BroadPhaseLayer,
	ShouldObjectLayersCollide []bool, // TODO: change this to only store/specify the lower triangle
) *System {
	return (*System)(C.newPhysics(
		C.int(BroadPhaseLayerCount),
		C.int(ObjectLayerCount),
		(*C.uint8_t)(unsafe.SliceData(ObjectLayerToBroadPhaseLayer)),
		(*C.bool)(unsafe.SliceData(ShouldObjectLayersCollide))))
}

type QueryFilter struct {
	// TODO: specify stuff for the body we pretend to be, i.e. its layer as well
	// as BodyID and list of other BodyIDs to ignore
	Ignore BodyID
}

type QueryHit struct {
	Point  geometry.DVec3
	Normal geometry.Vec3
	Depth  float32
}

type CastQueryResult struct {
	Fraction float32
	QueryHit
}

func (system *System) QueryShape(shape *Shape, pos geometry.DVec3, rot geometry.Rot3, scale geometry.Vec3, movementDirection geometry.Vec3, maxSeparationDistance float32, filter QueryFilter, hits []QueryHit) int {
	return int(C.physicsQueryShape(
		(*C.Physics)(system),
		(*C.Shape)(shape),
		(*C.dvec3)(unsafe.Pointer(&pos)),
		(*C.Rot3)(unsafe.Pointer(&rot)),
		(*C.vec3)(unsafe.Pointer(&scale)),
		(*C.vec3)(unsafe.Pointer(&movementDirection)),
		C.float(maxSeparationDistance),
		C.QueryFilter{ignore: C.BodyID(filter.Ignore)},
		(*C.QueryHit)(unsafe.Pointer(unsafe.SliceData(hits))),
		C.size_t(len(hits))))
}

func (system *System) QuerySweptShapeClosestHit(shape *Shape, pos geometry.DVec3, rot geometry.Rot3, scale geometry.Vec3, displacement geometry.Vec3, filter QueryFilter) CastQueryResult {
	result := C.physicsQuerySweptShapeClosestHit(
		(*C.Physics)(system),
		(*C.Shape)(shape),
		(*C.dvec3)(unsafe.Pointer(&pos)),
		(*C.Rot3)(unsafe.Pointer(&rot)),
		(*C.vec3)(unsafe.Pointer(&scale)),
		(*C.vec3)(unsafe.Pointer(&displacement)),
		C.QueryFilter{ignore: C.BodyID(filter.Ignore)})
	return *(*CastQueryResult)(unsafe.Pointer(&result))
}

func (system *System) SetGravity(gravity geometry.Vec3) {
	C.physicsSetGravity((*C.Physics)(system), (*C.vec3)(unsafe.Pointer(&gravity)))
}

func (system *System) Update(dt float32) {
	C.physicsUpdate((*C.Physics)(system), C.float(dt))
}

func (system *System) AddBody(bodyID BodyID, shape *Shape, pos geometry.DVec3, rot geometry.Rot3, vel, angVel geometry.Vec3, objectLayer int, ignoreBodyIDs []BodyID, motionType int, gravityFactor float32, mass float32, inertia geometry.Mat4x4) {
	motionProperties := C.MotionProperties{
		shape: (*C.Shape)(shape),
		motionState: C.MotionState{
			position:        *(*C.dvec3)(unsafe.Pointer(&pos)),
			rotation:        *(*C.Rot3)(unsafe.Pointer(&rot)),
			velocity:        *(*C.vec3)(unsafe.Pointer(&vel)),
			angularVelocity: *(*C.vec3)(unsafe.Pointer(&angVel)),
		},
		objectLayer:     C.int(objectLayer),
		ignoreBodies:    (*C.BodyID)(unsafe.Pointer(unsafe.SliceData(ignoreBodyIDs))),
		ignoreBodyCount: C.size_t(len(ignoreBodyIDs)),
		motionType:      C.int(motionType),
		gravityFactor:   C.float(gravityFactor),
		mass:            C.float(mass),
		inertia:         *(*C.mat4)(unsafe.Pointer(&inertia)),
	}
	C.physicsAddBody((*C.Physics)(system), C.BodyID(bodyID), motionProperties)
}

func (system *System) UpdateBody(bodyID BodyID, shape *Shape, pos geometry.DVec3, rot geometry.Rot3, vel, angVel geometry.Vec3, objectLayer int, ignoreBodyIDs []BodyID, motionType int, gravityFactor float32, mass float32, inertia geometry.Mat4x4) {
	motionProperties := C.MotionProperties{
		shape: (*C.Shape)(shape),
		motionState: C.MotionState{
			position:        *(*C.dvec3)(unsafe.Pointer(&pos)),
			rotation:        *(*C.Rot3)(unsafe.Pointer(&rot)),
			velocity:        *(*C.vec3)(unsafe.Pointer(&vel)),
			angularVelocity: *(*C.vec3)(unsafe.Pointer(&angVel)),
		},
		objectLayer:     C.int(objectLayer),
		ignoreBodies:    (*C.BodyID)(unsafe.Pointer(unsafe.SliceData(ignoreBodyIDs))),
		ignoreBodyCount: C.size_t(len(ignoreBodyIDs)),
		motionType:      C.int(motionType),
		gravityFactor:   C.float(gravityFactor),
		mass:            C.float(mass),
		inertia:         *(*C.mat4)(unsafe.Pointer(&inertia)),
	}
	C.physicsUpdateBody((*C.Physics)(system), C.BodyID(bodyID), motionProperties)
}

func (system *System) RemoveBody(bodyID BodyID) {
	C.physicsRemoveBody((*C.Physics)(system), C.BodyID(bodyID))
}

func (system *System) WritebackBody(bodyID BodyID) (pos geometry.DVec3, rot geometry.Rot3, vel, angVel geometry.Vec3) {
	var motionState C.MotionState
	C.physicsWritebackBody((*C.Physics)(system), C.BodyID(bodyID), &motionState)
	return *(*geometry.DVec3)(unsafe.Pointer(&motionState.position)), *(*geometry.Rot3)(unsafe.Pointer(&motionState.rotation)), *(*geometry.Vec3)(unsafe.Pointer(&motionState.velocity)), *(*geometry.Vec3)(unsafe.Pointer(&motionState.angularVelocity))
}

func (system *System) ActiveBodies() []BodyID {
	var cActiveBodyCount C.size_t
	cActiveBodies := C.physicsActiveBodies((*C.Physics)(system), &cActiveBodyCount)
	return unsafe.Slice((*BodyID)(unsafe.Pointer(cActiveBodies)), int(cActiveBodyCount))
}

func (system *System) ContactEvents() []ContactEvent {
	// TODO: if we're careful, we should get away with just pointer casting alone

	var cContactEventCount C.size_t
	cContactEvents := C.physicsContactEvents((*C.Physics)(system), &cContactEventCount)

	contactEvents := make([]ContactEvent, int(cContactEventCount))
	for i, cContactEvent := range unsafe.Slice(cContactEvents, int(cContactEventCount)) {
		contactEvents[i] = ContactEvent{
			Type: int32(cContactEvent._type),
			Body1: PerBodyContactData{
				BodyID:     BodyID(cContactEvent.body1.bodyID),
				SubShapeID: uint32(cContactEvent.body1.subShapeID),
				Active:     bool(cContactEvent.body1.active),
			},
			Body2: PerBodyContactData{
				BodyID:     BodyID(cContactEvent.body2.bodyID),
				SubShapeID: uint32(cContactEvent.body2.subShapeID),
				Active:     bool(cContactEvent.body2.active),
			},
			Normal: *(*geometry.Vec3)(unsafe.Pointer(&cContactEvent.normal)),
		}
	}

	return contactEvents
}
