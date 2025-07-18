package renderer

import (
	"encoding/binary"
	"io"
	"math"
	"sync"
	"unsafe"

	"github.com/go-json-experiment/json"

	"worldspawn/geometry-go"
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

// TODO: should we move deforming geometry into its own type? Or perhaps
// alternatively we should split off positions and normals from everything else.
type Mesh struct {
	rest *Mesh

	// TODO: remove this when we move file format parsing and handling elsewhere
	VertexGroups []string

	BLAS gpu.UnsafePointer

	groupElementsPerVertex uint32

	positions    gpu.Slice[[3]float32]
	normals      gpu.Slice[[3]float32]
	groupIndices gpu.Slice[uint32]
	groupWeights gpu.Slice[float32]
	uvs          gpu.Slice[[2]float32] // TODO: change this to be slice of pointers and call the thing attributes
	vertexCount  uint32

	// TODO: we may need different types of indices, how should we approach
	// that? Use interface{} or gpu.Slice[byte]?
	indexType      uint8
	primitives     gpu.Slice[[3]uint16]
	primitiveCount uint32

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

	blasConfig := gpu.AccelBuildConfig{
		Type: vk.ACCELERATION_STRUCTURE_TYPE_BOTTOM_LEVEL_KHR,
		Inputs: []gpu.AccelBuildInput{
			&gpu.AccelBuildInputTriangles{
				VertexFormat:   vk.FORMAT_R32G32B32_SFLOAT,
				VertexStride:   int(unsafe.Sizeof(m.positions.Value()[0])),
				MaxVertex:      uint32(max(gpu.SliceLen(m.positions)-1, 0)),
				IndexType:      vk.INDEX_TYPE_UINT16,
				PrimitiveCount: uint32(gpu.SliceLen(m.primitives)),
			},
		},
	}

	blasSize, _, _ := blasConfig.CalcSizes()

	blas := gpu.UnsafePointer(gpu.SliceData(gpu.MakeSliceUncached[byte](blasSize)))

	m.BLAS = blas
}

// TODO: should deformed meshes have their own type?
func (m *Mesh) InitDeforming(rest *Mesh) {
	*m = *rest
	m.rest = rest
}

func (m *Mesh) Rest() *Mesh {
	if m == nil {
		return nil
	}
	return m.rest
}

func divCeil(x, y int) int {
	return (x + y - 1) / y
}

// TODO: something is broken with deforming meshes, we get artifacts during trace

func (m *Mesh) EnqueueDeform(jq *gpu.JobQueue, pose gpu.Slice[geometry.Mat4x4]) {
	if m.positions == m.rest.positions {
		// TODO: do single big allocations for stuff with the same lifetime.
		m.positions = gpu.MakeSliceUncached[[3]float32](int(m.rest.vertexCount))
		m.normals = gpu.MakeSliceUncached[[3]float32](int(m.rest.vertexCount))
	}

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
		GroupElementsPerVertex: m.rest.groupElementsPerVertex,
		VertexCount:            m.rest.vertexCount,
		Pose:                   gpu.UnsafePointer(gpu.SliceData(pose)),
	}
	gpu.EnqueueParallelFor(jq, divCeil(int(m.rest.vertexCount), 64), skinMesh(), &args)

	m.buildBLAS(jq)
}

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
	blasConfig := &gpu.AccelBuildConfig{
		Type: vk.ACCELERATION_STRUCTURE_TYPE_BOTTOM_LEVEL_KHR,
		Inputs: []gpu.AccelBuildInput{
			&gpu.AccelBuildInputTriangles{
				VertexFormat:   vk.FORMAT_R32G32B32_SFLOAT,
				VertexBuffer:   gpu.UnsafePointer(gpu.SliceData(m.positions)),
				VertexStride:   int(unsafe.Sizeof(m.positions.Value()[0])),
				MaxVertex:      uint32(max(gpu.SliceLen(m.positions)-1, 0)),
				IndexType:      vk.INDEX_TYPE_UINT16,
				IndexBuffer:    gpu.UnsafePointer(gpu.SliceData(m.primitives)),
				PrimitiveCount: uint32(gpu.SliceLen(m.primitives)),
			},
		},
	}

	// TODO: we can cache blasSizes and blasConfig as well I think? Not
	// like it matters anyway because all of the code in this entire
	// file needs to be gutted.
	accelerationStructureSize, buildScratchSize, _ := blasConfig.CalcSizes()

	blasBuildScratch := gpu.UnsafePointer(gpu.SliceData(gpu.MakeSliceUncached[byte](buildScratchSize)))
	defer jq.Cleanup(func() { gpu.Free(blasBuildScratch) })
	gpu.EnqueueAccelBuild(jq,
		m.BLAS,
		accelerationStructureSize,
		blasConfig,
		blasBuildScratch)
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
		Header Section
		Blob   Section
	}

	type Part struct {
		VertexPositions    int64
		VertexNormals      int64
		VertexGroupIndices int64
		VertexGroupWeights int64
		VertexAttributes   []int64
		VertexCount        uint32

		IndexType      string
		Indices        int64
		PrimitiveCount uint32 // put a comment why we use prim count instead of index count.
	}

	type AttributeDesc struct {
		Name string
		Type string
	}

	type GeometryHeader struct {
		Type      string
		Rendering struct {
			VertexGroups                 []string
			VertexGroupElementsPerVertex int
			VertexAttributes             []AttributeDesc
			Parts                        []Part
		}
	}

	var preamble Preamble // TODO: rename to preamble or something
	if err := binary.Read(io.NewSectionReader(r, 0, math.MaxInt64), binary.LittleEndian, &preamble); err != nil {
		return err
	}

	// TODO: stringify numbers
	var header2 GeometryHeader
	if err := json.UnmarshalRead(io.NewSectionReader(r, preamble.Header.Off, preamble.Header.Size), &header2, json.StringifyNumbers(true)); err != nil {
		return err
	}

	blob2 := io.NewSectionReader(r, preamble.Blob.Off, preamble.Blob.Size)

	part := &header2.Rendering.Parts[0]

	m.VertexGroups = header2.Rendering.VertexGroups
	m.Init(2, part.PrimitiveCount, uint32(header2.Rendering.VertexGroupElementsPerVertex), part.VertexCount)

	var jq gpu.JobQueue
	err := m.SetFromFile(&jq,
		blob2,
		part.Indices,
		part.VertexPositions, part.VertexNormals,
		part.VertexGroupIndices,
		part.VertexGroupWeights,
		part.VertexAttributes[0])
	jq.WaitForIdle()

	return err
}

func byteslice[T any](s []T) []byte {
	sizeofT := int(unsafe.Sizeof(*new(T)))
	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(s))), len(s)*sizeofT)
}
