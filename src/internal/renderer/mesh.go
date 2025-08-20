package renderer

import (
	"encoding/binary"
	"io"
	"math"
	"math/rand/v2"
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
//   - support for arbitrary attributes (done)
//
//   - eventually we'll want to look into support for LODs

type IndexSlice struct {
	data gpu.UnsafePointer
	aux  uint64 // TODO: rename to reflect that it's packed index type and length
}

type MeshPart struct {
	PosBuffer     gpu.Slice[[3]float32]
	NormalBuffer  gpu.Slice[[3]float32]
	AttribBuffers []any
	VertexCount   int
	IndexBuffer   gpu.Slice[[3]uint16]
}

// TODO: should we move deforming geometry into its own type? Or perhaps
// alternatively we should split off positions and normals from everything else.
type Mesh struct {
	// TODO: remove this when we move file format parsing and handling elsewhere
	// VertexGroups []string

	// Belongs to the user's wrapper around Mesh.
	DefaultMaterials []MaterialInstance

	Accel gpu.Accel

	accelBuildConfig *gpu.AccelBuildConfig

	parts []MeshPart

	// attributes map[string]int
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
/*
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
*/

func (m *Mesh) buildAccel(jq *gpu.JobQueue) {
	m.Accel.EnqueueBuild(jq, m.accelBuildConfig)
}

func (m *Mesh) InitFromFile(r io.ReaderAt, filename string) error {
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
		PosBuffer     int64
		NormalBuffer  int64
		AttribBuffers []int64
		VertexCount   int
		IndexType     string
		IndexBuffer   int64
		TriangleCount int
	}

	type AttributeDesc struct {
		Name string
		Type string
	}

	type Rendering struct {
		Attributes []AttributeDesc
		Parts      []Part
	}

	type GeometryHeader struct {
		Rendering Rendering
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

	m.DefaultMaterials = make([]MaterialInstance, len(header2.Rendering.Parts))
	m.parts = make([]MeshPart, len(header2.Rendering.Parts))

	accelBuildInputs := make([]gpu.AccelBuildInput, len(header2.Rendering.Parts))
	for i, serializedPart := range header2.Rendering.Parts {
		part := &m.parts[i]

		part.PosBuffer = gpu.MakeSliceUncached[[3]float32](serializedPart.VertexCount)
		part.NormalBuffer = gpu.MakeSliceUncached[[3]float32](serializedPart.VertexCount)
		part.AttribBuffers = []any{
			gpu.MakeSliceUncached[[2]float32](serializedPart.VertexCount),
		}

		part.IndexBuffer = gpu.MakeSliceUncached[[3]uint16](serializedPart.TriangleCount)

		if _, err := blob2.ReadAt(byteslice(part.PosBuffer.Value()), serializedPart.PosBuffer); err != nil {
			panic("bug")
		}
		if _, err := blob2.ReadAt(byteslice(part.NormalBuffer.Value()), serializedPart.NormalBuffer); err != nil {
			panic("bug")
		}
		if _, err := blob2.ReadAt(byteslice(part.AttribBuffers[0].(gpu.Slice[[2]float32]).Value()), serializedPart.AttribBuffers[0]); err != nil {
			panic("bug")
		}

		if _, err := blob2.ReadAt(byteslice(part.IndexBuffer.Value()), serializedPart.IndexBuffer); err != nil {
			panic("bug")
		}

		m.DefaultMaterials[i] = MaterialInstance{
			Material: TestMaterial(),
			Hmm:      [3]float32{rand.Float32(), rand.Float32(), rand.Float32()},
		}

		accelBuildInputs[i] = &gpu.AccelBuildInputTriangles{
			VertexFormat:  vk.FORMAT_R32G32B32_SFLOAT,
			VertexBuffer:  gpu.UnsafePointer(gpu.SliceData(part.PosBuffer)),
			VertexCount:   gpu.SliceLen(part.PosBuffer),
			VertexStride:  int(unsafe.Sizeof(part.PosBuffer.Value()[0])),
			IndexType:     vk.INDEX_TYPE_UINT16, // TODO: infer from type of d.IndexBuffer
			IndexBuffer:   gpu.UnsafePointer(gpu.SliceData(part.IndexBuffer)),
			TriangleCount: gpu.SliceLen(part.IndexBuffer),
		}
	}

	m.accelBuildConfig = &gpu.AccelBuildConfig{
		Type:   vk.ACCELERATION_STRUCTURE_TYPE_BOTTOM_LEVEL_KHR,
		Inputs: accelBuildInputs,
	}

	m.Accel = gpu.NewAccel(m.accelBuildConfig)

	var jq gpu.JobQueue
	m.buildAccel(&jq)
	jq.WaitForIdle()

	return nil
}

func byteslice[T any](s []T) []byte {
	sizeofT := int(unsafe.Sizeof(*new(T)))
	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(s))), len(s)*sizeofT)
}
