package physics

// #include "physics.h"
import "C"

import (
	"runtime/cgo"
	"structs"
	"unsafe"

	"worldspawn/internal/gmath"
)

type SceneIntersection[T any] struct {
	_        structs.HostLayout
	BodyID   BodyID
	Geometry T
}

// TODO: shorter name pls
type QueryPipelineControl int

// TODO: reorder these or renumber these. Make IgnoreHit be -1?
const (
	Terminate QueryPipelineControl = iota
	AcceptHit
	IgnoreHit
)

// TODO: rename this to SceneQueryHitCollector
type SceneQueryPipeline[T any] interface {
	// TODO: don't require query pipeline to implement filters, be opportunistic
	// about it
	// FilterLayer(int) bool // TODO:  this feels like it should be a bitmap tbh
	// FilterBody(BodyID) bool
	Hit(SceneIntersection[T]) QueryPipelineControl // TODO: introduce a proper enum pls
}

//export physicsFilterLayerImpl
func physicsFilterLayerImpl(cpipeline unsafe.Pointer, layer uint32) C.bool {
	type Interface interface {
		FilterLayer(int) bool
	}

	if pipeline, ok := cgo.Handle(cpipeline).Value().(Interface); ok {
		return C.bool(pipeline.FilterLayer(int(layer)))
	}
	return true
}

//export physicsFilterBodyImpl
func physicsFilterBodyImpl(cpipeline unsafe.Pointer, bodyID BodyID) C.bool {
	type Interface interface {
		FilterBody(BodyID) bool
	}

	if pipeline, ok := cgo.Handle(cpipeline).Value().(Interface); ok {
		return C.bool(pipeline.FilterBody(BodyID(bodyID)))
	}
	return true
}

type Ray struct {
	_         structs.HostLayout
	Origin    gmath.Vec3f64
	Direction gmath.Vec3f32
	TMax      float32
}

// TODO: rename
func (r Ray) F(t float32) gmath.Vec3f64 {
	return r.Origin.Add(gmath.Vec3Convert[float64](r.Direction.Scale(t)))
}

type SceneRayHit = SceneIntersection[RayHit]

type RayHit struct {
	_ structs.HostLayout
	T float32
}

// TODO: rename to RayQuery?
func (system *System) TraceRay(ray Ray, pipeline SceneQueryPipeline[RayHit]) {
	cpipeline := cgo.NewHandle(pipeline)
	defer cpipeline.Delete()

	C.physicsTraceRay((*C.Physics)(system), *(*C.Ray)(unsafe.Pointer(&ray)), unsafe.Pointer(cpipeline))
}

//export physicsRayHitImpl
func physicsRayHitImpl(pipeline unsafe.Pointer, hit C.SceneRayHit) C.int {
	return C.int(cgo.Handle(pipeline).Value().(SceneQueryPipeline[RayHit]).Hit(*(*SceneRayHit)(unsafe.Pointer(&hit))))
}

type Overlap struct {
	_     structs.HostLayout
	Pos   gmath.Vec3f64
	Rot   gmath.Rot3
	Scale gmath.Vec3f32
	Shape *Shape

	MovementDirection     gmath.Vec3f32
	MaxSeparationDistance float32
}

type OverlapHit struct {
	_                structs.HostLayout
	ContactPointOn1  gmath.Vec3f32
	ContactPointOn2  gmath.Vec3f32
	PenetrationAxis  gmath.Vec3f32
	PenetrationDepth float32
	// TODO: more stuff?
}

// TODO: rename to GeometryIntersectionQuery or idk?
func (system *System) Overlap(overlap Overlap, pipeline SceneQueryPipeline[OverlapHit]) {
	cpipeline := cgo.NewHandle(pipeline)
	defer cpipeline.Delete()

	C.physicsOverlapQuery((*C.Physics)(system), *(*C.Overlap)(unsafe.Pointer(&overlap)), unsafe.Pointer(cpipeline))
}

//export physicsOverlapHitTramp
func physicsOverlapHitTramp(pipeline unsafe.Pointer, hit C.SceneOverlapHit) C.int {
	return C.int(cgo.Handle(pipeline).Value().(SceneQueryPipeline[OverlapHit]).Hit((*(*SceneIntersection[OverlapHit])(unsafe.Pointer(&hit)))))
}
