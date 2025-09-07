package physics

//go:generate stringer -trimprefix Shape -type ShapeKind -output shape_kind_string.go

// #include "c/physics.h"
import "C"

import (
	"encoding/binary"
	"io"
	"io/fs"
	"unsafe"

	"github.com/go-json-experiment/json"

	"worldspawn/geometry-go"
)

// TODO: pass materials

func NewSphereShape(radius float32) (*Shape, error) {
	return (*Shape)(C.newSphereShape(C.float(radius))), nil
}

func NewBoxShape(halfExtent geometry.Vec3, convexRadius float32) (*Shape, error) {
	return (*Shape)(C.newBoxShape((*C.vec3)(unsafe.Pointer(&halfExtent)), C.float(convexRadius))), nil
}

func NewCylinderShape(radius, halfHeight, convexRadius float32) (*Shape, error) {
	return (*Shape)(C.newCylinderShape(C.float(radius), C.float(halfHeight), C.float(convexRadius))), nil
}

func NewConvexHullShape(vertices []geometry.Vec3, convexRadius float32) (*Shape, error) {
	return (*Shape)(C.newConvexHullShape(
		(*C.vec3)(unsafe.Pointer(unsafe.SliceData(vertices))), C.size_t(len(vertices)),
		C.float(convexRadius))), nil
}

func NewMeshShape(vertices []geometry.Vec3, triangles []Triangle) (*Shape, error) {
	// HACK: remove this when we plumb materials
	for i := range triangles {
		triangles[i].MaterialIndex = 0
	}
	return (*Shape)(C.newMeshShape(
		(*C.vec3)(unsafe.Pointer(unsafe.SliceData(vertices))), C.size_t(len(vertices)),
		unsafe.Pointer(unsafe.SliceData(triangles)), C.size_t(len(triangles)))), nil
}

func NewTransformedShape(translation geometry.Vec3, rotation geometry.Rot3, scale geometry.Vec3, shape *Shape) (*Shape, error) {
	return (*Shape)(C.newTransformedShape((*C.vec3)(unsafe.Pointer(&translation)), (*C.Rot3)(unsafe.Pointer(&rotation)), (*C.vec3)(unsafe.Pointer(&scale)), (*C.Shape)(unsafe.Pointer(shape)))), nil
}

// TODO: move this into a file separate from other shapes, probably
func NewFileBackedShape(fsys fs.FS, filename string, concave bool) (*Shape, error) {
	f, err := fsys.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var preamble struct {
		Magic  [16]byte
		Magic2 [16]byte
		A, B   struct {
			Off, Len int64
		}
	}
	if err := binary.Read(f, binary.LittleEndian, &preamble); err != nil {
		return nil, err
	}

	rat := f.(io.ReaderAt)

	var header2 Header2
	if err := json.UnmarshalRead(io.NewSectionReader(rat, preamble.A.Off, preamble.A.Len), &header2, json.StringifyNumbers(true)); err != nil {
		return nil, err
	}

	shape := header2.Collision

	blob := io.NewSectionReader(rat, preamble.B.Off, preamble.B.Len)

	verts := make([]geometry.Vec3, shape.VertexCount)
	blob.Seek(int64(shape.VertexBuffer), io.SeekStart)
	if err := binary.Read(blob, binary.LittleEndian, &verts); err != nil {
		return nil, err
	}

	tris := make([]Triangle, shape.TriangleCount)
	blob.Seek(int64(shape.TriangleBuffer), io.SeekStart)
	if err := binary.Read(blob, binary.LittleEndian, &tris); err != nil {
		return nil, err
	}

	if concave {
		return NewMeshShape(verts, tris)
	} else {
		return NewConvexHullShape(verts, 0.05)
	}
}

func (s *Shape) Mass() float32 {
	return float32(C.shapeMass((*C.Shape)(s)))
}

func (s *Shape) Inertia() geometry.Mat4x4 {
	m := C.shapeInertia((*C.Shape)(s))
	return *(*geometry.Mat4x4)(unsafe.Pointer(&m))
}

// TODO: rename
func (s *Shape) DecRef() {
	C.shapeDecRef((*C.Shape)(s))
}
