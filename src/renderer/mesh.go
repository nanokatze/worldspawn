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

/*
type IndexType uint8

const (
	IndexTypeNone IndexType = iota
	IndexTypeUint8
	IndexTypeUint16
	IndexTypeUint32
)
*/

/*
type GPUFormattedSlice struct {
	data   gpu.UnsafePointer
	packed uint64
}

func formattedSliceLen(s GPUFormattedSlice) int {
	return int(s.packed & ((1 << 57) - 1))
}
*/

type MeshPart struct {
	Positions   gpu.Slice[[3]float32]
	Normals     gpu.Slice[[3]float32]
	Attributes  []any
	VertexCount int

	Triangles gpu.Slice[[3]uint16]
	IndexType uint8
}

// TODO: should we move deforming geometry into its own type? Or perhaps
// alternatively we should split off positions and normals from everything else.
type Mesh struct {
	// TODO: remove this when we move file format parsing and handling elsewhere
	// VertexGroups []string

	Accel gpu.Accel

	accelBuildConfig *gpu.AccelBuildConfig

	parts []MeshPart

	attributes map[string]int
}

// func NewMesh()

func (m *Mesh) Init(indexType uint8, primitiveCount uint32, groupElementsPerVertex uint32, vertexCount uint32) {
	part := MeshPart{}
	part.IndexType = indexType

	// TODO: do single big allocations for stuff with the same lifetime

	switch indexType {
	case 2:
		part.Triangles = gpu.MakeSliceUncached[[3]uint16](int(primitiveCount))

	default:
		panic("TODO/unknown index type")
	}

	part.Positions = gpu.MakeSliceUncached[[3]float32](int(vertexCount))
	part.Normals = gpu.MakeSliceUncached[[3]float32](int(vertexCount))
	part.Attributes = []any{
		gpu.MakeSliceUncached[[2]float32](int(vertexCount)),
	}

	m.parts = []MeshPart{part}

	m.accelBuildConfig = &gpu.AccelBuildConfig{
		Type: vk.ACCELERATION_STRUCTURE_TYPE_BOTTOM_LEVEL_KHR,
		Inputs: []gpu.AccelBuildInput{
			&gpu.AccelBuildInputTriangles{
				VertexBuffer:   gpu.UnsafePointer(gpu.SliceData(part.Positions)),
				VertexCount:    gpu.SliceLen(part.Positions),
				VertexStride:   int(unsafe.Sizeof(part.Positions.Value()[0])),
				VertexFormat:   vk.FORMAT_R32G32B32_SFLOAT,
				TriangleBuffer: gpu.UnsafePointer(gpu.SliceData(part.Triangles)),
				TriangleCount:  gpu.SliceLen(part.Triangles),
				IndexType:      vk.INDEX_TYPE_UINT16,
			},
		},
	}

	m.Accel = gpu.NewAccel(m.accelBuildConfig)
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
	part := m.parts[0]
	if _, err := r.ReadAt(byteslice(part.Triangles.Value()), indicesOff); err != nil {
		panic("bug")
	}
	if _, err := r.ReadAt(byteslice(part.Positions.Value()), positionsOff); err != nil {
		panic("bug")
	}
	if _, err := r.ReadAt(byteslice(part.Normals.Value()), normalsOff); err != nil {
		panic("bug")
	}
	if _, err := r.ReadAt(byteslice(part.Attributes[0].(gpu.Slice[[2]float32]).Value()), uvsOff); err != nil {
		panic("bug")
	}

	m.buildAccel(jq)

	return nil
}

func (m *Mesh) buildAccel(jq *gpu.JobQueue) {
	m.Accel.EnqueueBuild(jq, m.accelBuildConfig)
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
		Magic2 [16]byte // "Geometry"
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
