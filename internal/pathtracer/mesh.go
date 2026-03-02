package pathtracer

import (
	"unsafe"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

// TODO: make this gpu-accessible
// TODO: rename
type MeshPart struct {
	IndexBuffer      gpu.Slice[[3]uint16]
	AttributeBuffers []any
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

	PositionAttribute int
	NormalAttribute   int

	// TODO: deinterleave so that we have an array of attribute and index buffers
	Parts []MeshPart

	// TODO: outline the following fields into a separate struct

	accelBuildConfig *gpu.AccelBuildConfig
	accel            gpu.Accel
}

/*
type MeshWithAccel struct {
	accel gpu.Accel
	mesh  *Mesh
}
*/

// TODO: remove this and replace with MeshWithAccel and appropriate constructor.
// We might need some adjustments for this to work with cluster accel.
func (m *Mesh) InitAccel() {
	accelBuildInputs := make([]gpu.AccelBuildInput, len(m.Parts))
	for i, part := range m.Parts {
		positionBuffer := part.AttributeBuffers[m.PositionAttribute].(gpu.Slice[[3]float32])

		accelBuildInputs[i] = &gpu.AccelBuildInputTriangles{
			VertexFormat:  vk.FORMAT_R32G32B32_SFLOAT,
			VertexBuffer:  gpu.UnsafePointer(gpu.SliceData(positionBuffer)),
			VertexCount:   gpu.SliceLen(positionBuffer),
			VertexStride:  int(unsafe.Sizeof(positionBuffer.Value()[0])),
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
