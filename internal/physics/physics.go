package physics

// #cgo CXXFLAGS: -std=c++20
// #cgo CXXFLAGS: -DJPH_CROSS_PLATFORM_DETERMINISTIC -DJPH_DOUBLE_PRECISION -DJPH_ENABLE_ASSERTS -DJPH_OBJECT_STREAM -DJPH_USE_AVX -DJPH_USE_AVX2 -DJPH_USE_CPU_COMPUTE -DJPH_USE_F16C -DJPH_USE_LZCNT -DJPH_USE_SSE4_1 -DJPH_USE_SSE4_2 -DJPH_USE_TZCNT
// #cgo CXXFLAGS: -mavx2 -mfpmath=sse
// #cgo LDFLAGS: -lJolt
// #cgo LDFLAGS: -lm -lstdc++
//
// #include "physics.h"
import "C"

import (
	"slices"
	"unsafe"

	"worldspawn/internal/gmath"
)

// TODO: replace with a plain int
type BodyID uint32

// TODO: rename to Scene?
type System C.Physics

// TODO: rename to Geometry
type Shape C.Shape

type Triangle struct {
	VertexIndices [3]uint32
	MaterialIndex uint32
}

// TODO: hide the internals and require that the user uses Set
type LayerCollisionRules []bool

func (layers *LayerCollisionRules) Set(a, b int) {
	panic("not implemented")
}

type ContactListener interface {
	ContactValidate() // TODO: rename to ShouldCollide?
	ContactAdded()
	ContactRemoved()
}

func NewSystem(
	ObjectLayerCount int,
	ObjectLayerToBroadPhaseLayer []uint8, // TODO: use plain int here?
	layerRules LayerCollisionRules,
) *System {
	if len(layerRules) != ObjectLayerCount*(ObjectLayerCount+1)/2 {
		panic("pls fix your rules")
	}

	BroadPhaseLayerCount := int(slices.Max(ObjectLayerToBroadPhaseLayer)) + 1

	return (*System)(C.newPhysics(
		C.int(BroadPhaseLayerCount),
		C.int(ObjectLayerCount),
		(*C.uint8_t)(unsafe.SliceData(ObjectLayerToBroadPhaseLayer)),
		(*C.bool)(unsafe.SliceData(layerRules))))
}

// TODO: factor all parameters into a struct?
func (system *System) Update(dt float32, gravity gmath.Vec3f32) {
	C.physicsSetGravity((*C.Physics)(system), *(*C.vec3)(unsafe.Pointer(&gravity)))
	C.physicsUpdate((*C.Physics)(system), C.float(dt))
}

func (system *System) AddBody(
	bodyID BodyID,
	shape *Shape,
	pos gmath.Vec3f64,
	rot gmath.Rot3,
	vel, angVel gmath.Vec3f32,
	objectLayer int,
	motionType int,
	gravityFactor float32,
	mass float32,
	inertia gmath.Mat4x4f32,
	sensor bool) {
	motionProperties := C.MotionProperties{
		shape: (*C.Shape)(shape),
		motionState: C.MotionState{
			position:        *(*C.dvec3)(unsafe.Pointer(&pos)),
			rotation:        *(*C.Rot3)(unsafe.Pointer(&rot)),
			velocity:        *(*C.vec3)(unsafe.Pointer(&vel)),
			angularVelocity: *(*C.vec3)(unsafe.Pointer(&angVel)),
		},
		objectLayer:   C.int(objectLayer),
		motionType:    C.int(motionType),
		gravityFactor: C.float(gravityFactor),
		mass:          C.float(mass),
		inertia:       *(*C.mat4)(unsafe.Pointer(&inertia)),
		sensor:        C.bool(sensor),
	}
	C.physicsAddBody((*C.Physics)(system), C.BodyID(bodyID), motionProperties)
}

// TODO: merge AddBody and UpdateBody into one
func (system *System) UpdateBody(
	bodyID BodyID,
	shape *Shape,
	pos gmath.Vec3f64,
	rot gmath.Rot3,
	vel, angVel gmath.Vec3f32,
	objectLayer int,
	motionType int,
	gravityFactor float32,
	mass float32,
	inertia gmath.Mat4x4f32,
	sensor bool) {
	motionProperties := C.MotionProperties{
		shape: (*C.Shape)(shape),
		motionState: C.MotionState{
			position:        *(*C.dvec3)(unsafe.Pointer(&pos)),
			rotation:        *(*C.Rot3)(unsafe.Pointer(&rot)),
			velocity:        *(*C.vec3)(unsafe.Pointer(&vel)),
			angularVelocity: *(*C.vec3)(unsafe.Pointer(&angVel)),
		},
		objectLayer:   C.int(objectLayer),
		motionType:    C.int(motionType),
		gravityFactor: C.float(gravityFactor),
		mass:          C.float(mass),
		inertia:       *(*C.mat4)(unsafe.Pointer(&inertia)),
		sensor:        C.bool(sensor),
	}
	C.physicsUpdateBody((*C.Physics)(system), C.BodyID(bodyID), motionProperties)
}

func (system *System) RemoveBody(bodyID BodyID) {
	C.physicsRemoveBody((*C.Physics)(system), C.BodyID(bodyID))
}

func (system *System) WritebackBody(bodyID BodyID) (pos gmath.Vec3f64, rot gmath.Rot3, vel, angVel gmath.Vec3f32) {
	var motionState C.MotionState
	C.physicsWritebackBody((*C.Physics)(system), C.BodyID(bodyID), &motionState)
	return *(*gmath.Vec3f64)(unsafe.Pointer(&motionState.position)), *(*gmath.Rot3)(unsafe.Pointer(&motionState.rotation)), *(*gmath.Vec3f32)(unsafe.Pointer(&motionState.velocity)), *(*gmath.Vec3f32)(unsafe.Pointer(&motionState.angularVelocity))
}

func (system *System) ActiveBodies() []BodyID {
	var cActiveBodyCount C.size_t
	cActiveBodies := C.physicsActiveBodies((*C.Physics)(system), &cActiveBodyCount)
	return unsafe.Slice((*BodyID)(unsafe.Pointer(cActiveBodies)), int(cActiveBodyCount))
}

// TODO: kill this in favor of just passing contact listener
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
			Normal: *(*gmath.Vec3f32)(unsafe.Pointer(&cContactEvent.normal)),
		}
	}

	return contactEvents
}
