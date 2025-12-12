package pathtracer

import (
	"unsafe"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

// TODO: make this gpu-accessible
// TODO: when we git gud don't have special PosBuffer, NormalBuffer etc slices?
type MeshPart struct {
	PosBuffer     gpu.Slice[[3]float32]
	NormalBuffer  gpu.Slice[[3]float32]
	AttribBuffers []any // TODO: rename to just Buffers eventually
	VertexCount   int
	IndexBuffer   gpu.Slice[[3]uint16]
}

// TODO: flatten our mesh representation, i.e. have a single long pool of
// positions and other attributes, and have mesh parts (rename to make it clear
// that it's just a bunch of tris with the same material index) just be offsets
// into the index buffer.
// TODO: come up with a solution to preserve authored material index? It would
// be nice so that blender materials can use material_index attribute without
// extra gymnastics...
type Mesh struct {
	// TODO: remove this when we move file format parsing and handling elsewhere
	// VertexGroups []string

	// We actually need this map once we stuff all attributes into a single
	// array of attribute descriptors.
	// attributes map[string]int

	Parts []MeshPart

	accelBuildConfig *gpu.AccelBuildConfig
	accel            gpu.Accel
}

// TODO: remove this and replace with MeshWithAccel and appropriate constructor.
func (m *Mesh) InitAccel() {
	accelBuildInputs := make([]gpu.AccelBuildInput, len(m.Parts))
	for i, part := range m.Parts {
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
	m.accel = gpu.NewAccel(m.accelBuildConfig)
}

// TODO: better naming
func (m *Mesh) BuildAccel(jq *gpu.JobQueue) {
	m.accel.EnqueueBuild(jq, m.accelBuildConfig)
}
