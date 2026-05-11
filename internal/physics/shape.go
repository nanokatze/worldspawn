package physics

// #include "physics.h"
import "C"

import (
	"encoding/binary"
	"io"
	"io/fs"
	"unsafe"

	"github.com/go-json-experiment/json"

	"worldspawn/internal/gmath"
	"worldspawn/internal/loaders/wmesh"
)

// TODO: pass materials

func NewSphereShape(radius float32) (*Shape, error) {
	return (*Shape)(C.newSphereShape(C.float(radius))), nil
}

func NewBoxShape(halfExtent gmath.Vec3f32, convexRadius float32) (*Shape, error) {
	return (*Shape)(C.newBoxShape((*C.vec3)(unsafe.Pointer(&halfExtent)), C.float(convexRadius))), nil
}

func NewCylinderShape(radius, halfHeight, convexRadius float32) (*Shape, error) {
	return (*Shape)(C.newCylinderShape(C.float(radius), C.float(halfHeight), C.float(convexRadius))), nil
}

func NewConvexHullShape(vertices []gmath.Vec3f32, convexRadius float32) (*Shape, error) {
	return (*Shape)(C.newConvexHullShape(
		(*C.vec3)(unsafe.Pointer(unsafe.SliceData(vertices))), C.size_t(len(vertices)),
		C.float(convexRadius))), nil
}

func NewMeshShape(vertices []gmath.Vec3f32, triangles []Triangle) (*Shape, error) {
	// HACK
	triangles2 := make([]struct {
		Triangle
		UserData uint32
	}, len(triangles))
	for i := range triangles {
		triangles2[i].Triangle = triangles[i]
		triangles2[i].Triangle.MaterialIndex = 0
	}
	return (*Shape)(C.newMeshShape(
		(*C.vec3)(unsafe.Pointer(unsafe.SliceData(vertices))), C.size_t(len(vertices)),
		unsafe.Pointer(unsafe.SliceData(triangles2)), C.size_t(len(triangles2)))), nil
}

func NewTransformedShape(translation gmath.Vec3f32, rotation gmath.Rot3, scale gmath.Vec3f32, shape *Shape) (*Shape, error) {
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

	var header2 wmesh.Header
	if err := json.UnmarshalRead(io.NewSectionReader(rat, preamble.A.Off, preamble.A.Len), &header2, json.StringifyNumbers(true)); err != nil {
		return nil, err
	}

	blob := io.NewSectionReader(rat, preamble.B.Off, preamble.B.Len)

	indexBuffer := make([][3]uint32, header2.PrimitiveCount)
	blob.Seek(header2.IndexBuffer.Data, io.SeekStart)
	if err := binary.Read(blob, binary.LittleEndian, &indexBuffer); err != nil {
		return nil, err
	}

	posBuffer := make([]gmath.Vec3f32, header2.VertexCount)
	blob.Seek(header2.Positions.Data.Data, io.SeekStart)
	if err := binary.Read(blob, binary.LittleEndian, &posBuffer); err != nil {
		return nil, err
	}

	tris := make([]Triangle, header2.PrimitiveCount)
	for i := range header2.PrimitiveCount {
		tris[i] = Triangle{
			VertexIndices: [3]uint32{
				uint32(indexBuffer[i][0]),
				uint32(indexBuffer[i][1]),
				uint32(indexBuffer[i][2]),
			},
		}
	}

	if concave {
		return NewMeshShape(posBuffer, tris)
	} else {
		return NewConvexHullShape(posBuffer, 0.05)
	}
}

func (s *Shape) Mass() float32 {
	return float32(C.shapeMass((*C.Shape)(s)))
}

func (s *Shape) Inertia() gmath.Mat4x4f32 {
	m := C.shapeInertia((*C.Shape)(s))
	return *(*gmath.Mat4x4f32)(unsafe.Pointer(&m))
}

// TODO: rename
func (s *Shape) DecRef() {
	C.shapeDecRef((*C.Shape)(s))
}
