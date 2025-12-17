package pathtracer

import (
	"unsafe"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

// TODO: make this gpu-accessible
type MeshPart struct {
	AttribBuffers []any
	IndexBuffer   gpu.Slice[[3]uint16]
}

// TODO: come up with a solution to preserve authored material index? It would
// be nice so that blender materials can use material_index attribute without
// extra gymnastics... I guess there's nothing we can do except a remap table :/
type Mesh struct {
	// TODO: attribute descs. We could then make MeshPart.AttributeBuffers be
	// "just pointers". If we make attribute descs also specify strides we would
	// let cookers interleave attributes as they please. The only gotcha is that
	// if we allow multiple attributes to live in the same buffer they all would
	// have to be in the same domain (per-vertex or per-triangle)

	// TODO: rename these?
	PosBuffer    int
	NormalBuffer int

	// TODO: deinterleave so that we have an array of attribute and index buffers
	Parts []MeshPart

	// TODO: outline the following fields into a separate struct

	accelBuildConfig *gpu.AccelBuildConfig
	accel            gpu.Accel
}

// TODO: remove this and replace with MeshWithAccel and appropriate constructor.
func (m *Mesh) InitAccel() {
	accelBuildInputs := make([]gpu.AccelBuildInput, len(m.Parts))
	for i, part := range m.Parts {
		posBuffer := part.AttribBuffers[m.PosBuffer].(gpu.Slice[[3]float32])

		accelBuildInputs[i] = &gpu.AccelBuildInputTriangles{
			VertexFormat:  vk.FORMAT_R32G32B32_SFLOAT,
			VertexBuffer:  gpu.UnsafePointer(gpu.SliceData(posBuffer)),
			VertexCount:   gpu.SliceLen(posBuffer),
			VertexStride:  int(unsafe.Sizeof(posBuffer.Value()[0])),
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
