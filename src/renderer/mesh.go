package renderer

import (
	"encoding/binary"
	"io"
	"math"
	"sync"
	"unsafe"

	"github.com/go-json-experiment/json"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

// TODO: we should be pushing as much as we can onto the user. Could we perhaps
// get rid of Mesh altogether? I don't think we can

var skinMesh = sync.OnceValue(func() *gpu.Func {
	return gpu.NewFunc(mustReadFile("shaders/mesh.spv"), vk.SHADER_STAGE_COMPUTE_BIT, "skinMesh")
})

// Features checklist:
//
//   - support instancing (recursive as well) so we can support use
//     cases like instanced sleepers on rail tracks. Blender's meshes
//     support recursive instancing and we can export them as is.
//
//   - support for skeletal deformations with very high number of bones (done)
//
//   - support for arbitrary attributes
//
//   - eventually we'll want to look into support for LODs

type meshPart struct {
	attributes []any
}

// TODO: should we move deforming geometry into its own type? Or perhaps
// alternatively we should split off positions and normals from everything else.
type Mesh struct {
	// TODO: remove this when we move file format parsing and handling elsewhere
	VertexGroups []string

	Accel gpu.Accel

	// TODO: we may need different types of indices, how should we approach
	// that? Use interface{} or gpu.Slice[byte]?
	indexType      uint8
	primitives     gpu.Slice[[3]uint16]
	primitiveCount uint32

	groupElementsPerVertex uint32

	positions    gpu.Slice[[3]float32]
	normals      gpu.Slice[[3]float32]
	groupIndices gpu.Slice[uint32]
	groupWeights gpu.Slice[float32]
	uvs          gpu.Slice[[2]float32] // TODO: change this to be slice of pointers and call the thing attributes
	vertexCount  uint32

	attributes map[string]any

	// parts []meshPart
}

func (m *Mesh) Init(indexType uint8, primitiveCount uint32, groupElementsPerVertex uint32, vertexCount uint32) {
	m.indexType = indexType
	m.primitiveCount = primitiveCount
	m.groupElementsPerVertex = groupElementsPerVertex
	m.vertexCount = vertexCount

	// TODO: do single big allocations for stuff with the same lifetime

	switch m.indexType {
	case 2:
		m.primitives = gpu.MakeSliceUncached[[3]uint16](int(primitiveCount))

	default:
		panic("TODO/unknown index type")
	}

	m.positions = gpu.MakeSliceUncached[[3]float32](int(vertexCount))
	m.normals = gpu.MakeSliceUncached[[3]float32](int(vertexCount))
	if groupElementsPerVertex > 0 {
		m.groupIndices = gpu.MakeSliceUncached[uint32](int(vertexCount) * int(groupElementsPerVertex))
		m.groupWeights = gpu.MakeSliceUncached[float32](int(vertexCount) * int(groupElementsPerVertex))
	}
	m.uvs = gpu.MakeSliceUncached[[2]float32](int(vertexCount))

	/*
		m.attributes = map[string]any{
			"UVMap": gpu.MakeSliceUncached[[2]float32](int(vertexCount)),
		}
	*/

	m.Accel = gpu.NewAccel(
		&gpu.AccelBuildConfig{
			Type: vk.ACCELERATION_STRUCTURE_TYPE_BOTTOM_LEVEL_KHR,
			Inputs: []gpu.AccelBuildInput{
				&gpu.AccelBuildInputTriangles{
					VertexCount:    gpu.SliceLen(m.positions),
					VertexStride:   int(unsafe.Sizeof(m.positions.Value()[0])),
					VertexFormat:   vk.FORMAT_R32G32B32_SFLOAT,
					PrimitiveCount: gpu.SliceLen(m.primitives),
					IndexType:      vk.INDEX_TYPE_UINT16,
				},
			},
		})
}

func divCeil(x, y int) int {
	return (x + y - 1) / y
}

// TODO: something is broken with deforming meshes, we get artifacts during trace

/*
func (dst *Mesh) EnqueueDeform(jq *gpu.JobQueue, src *Mesh, pose gpu.Slice[geometry.Mat4x4]) {
	args := struct {
		SkinnedPositions       gpu.UnsafePointer
		SkinnedNormals         gpu.UnsafePointer
		RestPositions          gpu.UnsafePointer
		RestNormals            gpu.UnsafePointer
		GroupIndices           gpu.UnsafePointer
		GroupWeights           gpu.UnsafePointer
		GroupElementsPerVertex uint32
		VertexCount            uint32
		Pose                   gpu.UnsafePointer
	}{
		SkinnedPositions:       gpu.UnsafePointer(gpu.SliceData(m.positions)),
		SkinnedNormals:         gpu.UnsafePointer(gpu.SliceData(m.normals)),
		RestPositions:          gpu.UnsafePointer(gpu.SliceData(m.rest.positions)),
		RestNormals:            gpu.UnsafePointer(gpu.SliceData(m.rest.normals)),
		GroupIndices:           gpu.UnsafePointer(gpu.SliceData(m.rest.groupIndices)),
		GroupWeights:           gpu.UnsafePointer(gpu.SliceData(m.rest.groupWeights)),
		GroupElementsPerVertex: src.groupElementsPerVertex,
		VertexCount:            src.vertexCount,
		Pose:                   gpu.UnsafePointer(gpu.SliceData(pose)),
	}
	gpu.EnqueueParallelFor(jq, divCeil(int(src.vertexCount), 64), skinMesh(), &args)

	m.buildBLAS(jq)
}
*/

// TODO: Mesh consists of several parts so we need to be able to provide all
// parts at once I guess.
//
// TODO: same as texture, this should work on device timeline. Eventually we
// might gdeflate-compress things, so we'll be decompressing on the device.
func (m *Mesh) SetFromFile(
	jq *gpu.JobQueue,

	r io.ReaderAt,

	indicesOff int64,

	positionsOff int64,
	normalsOff int64,

	groupIndicesOff int64,
	groupWeightsOff int64,

	uvsOff int64,
) error {
	if _, err := r.ReadAt(byteslice(m.primitives.Value()), indicesOff); err != nil {
		panic("bug")
	}
	if _, err := r.ReadAt(byteslice(m.positions.Value()), positionsOff); err != nil {
		panic("bug")
	}
	if _, err := r.ReadAt(byteslice(m.normals.Value()), normalsOff); err != nil {
		panic("bug")
	}
	if m.groupElementsPerVertex > 0 {
		if _, err := r.ReadAt(byteslice(m.groupIndices.Value()), groupIndicesOff); err != nil {
			panic("bug")
		}
		if _, err := r.ReadAt(byteslice(m.groupWeights.Value()), groupWeightsOff); err != nil {
			panic("bug")
		}
	}
	if _, err := r.ReadAt(byteslice(m.uvs.Value()), uvsOff); err != nil {
		panic("bug")
	}

	m.buildBLAS(jq)

	return nil
}

func (m *Mesh) buildBLAS(jq *gpu.JobQueue) {
	m.Accel.EnqueueBuild(jq,
		&gpu.AccelBuildConfig{
			Type: vk.ACCELERATION_STRUCTURE_TYPE_BOTTOM_LEVEL_KHR,
			Inputs: []gpu.AccelBuildInput{
				&gpu.AccelBuildInputTriangles{
					VertexBuffer:   gpu.UnsafePointer(gpu.SliceData(m.positions)),
					VertexCount:    gpu.SliceLen(m.positions),
					VertexStride:   int(unsafe.Sizeof(m.positions.Value()[0])),
					VertexFormat:   vk.FORMAT_R32G32B32_SFLOAT,
					IndexBuffer:    gpu.UnsafePointer(gpu.SliceData(m.primitives)),
					PrimitiveCount: gpu.SliceLen(m.primitives),
					IndexType:      vk.INDEX_TYPE_UINT16,
				},
			},
		})
}

// TODO: remove in favor of InitFromFile2
func (m *Mesh) InitFromFile(r io.ReaderAt) error {
	type Section struct {
		Off, Size int64
	}

	// TODO: move Preamble handling out of this package entirely
	//
	// NOTE: MDL and MDLg now live in the same file and share the structure.
	type Preamble struct {
		Magic  [16]byte // "Worldspawn"
		Magic2 [16]byte
		Header Section
		Blob   Section
	}

	type Part struct {
		Positions   int64
		Normals     int64
		Attributes  []int64
		VertexCount uint32

		IndexType     string
		Triangles     int64
		TriangleCount uint32
	}

	type AttributeDesc struct {
		Name string
		Type string
	}

	type GeometryHeader struct {
		Rendering []Part
	}

	var preamble Preamble // TODO: rename to preamble or something
	if err := binary.Read(io.NewSectionReader(r, 0, math.MaxInt64), binary.LittleEndian, &preamble); err != nil {
		return err
	}

	var header2 GeometryHeader
	if err := json.UnmarshalRead(io.NewSectionReader(r, preamble.Header.Off, preamble.Header.Size), &header2, json.StringifyNumbers(true)); err != nil {
		return err
	}

	blob2 := io.NewSectionReader(r, preamble.Blob.Off, preamble.Blob.Size)

	part := &header2.Rendering[0]

	// m.VertexGroups = header2.Rendering.VertexGroups
	m.Init(2, part.TriangleCount, uint32(0), part.VertexCount)

	var jq gpu.JobQueue
	err := m.SetFromFile(&jq,
		blob2,
		part.Triangles,
		part.Positions, part.Normals,
		0,
		0,
		part.Attributes[0])
	jq.WaitForIdle()

	return err
}

func byteslice[T any](s []T) []byte {
	sizeofT := int(unsafe.Sizeof(*new(T)))
	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(s))), len(s)*sizeofT)
}
