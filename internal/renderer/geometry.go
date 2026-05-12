package renderer

import (
	"unsafe"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

const (
	// TODO: reprefix these some other way?
	// TODO: we'll have different attribute indices depending on geometry type.
	// These are for meshes, but for e.g. curves things would be different.
	AttributePosition = iota
	AttributeNormal
)

// TODO: kill this structure?
type GeometryPart struct {
	IndexBuffer IndexBuffer
}

// TODO: come up with a solution to preserve authored material index? It would
// be nice so that blender materials can use material_index attribute without
// extra gymnastics... I guess there's nothing we can do except a remap table :/
type Geometry struct {
	// TODO: geometry type (triangle mesh, curves, etc)

	// TODO: attribute descs. We could then make MeshPart.AttributeBuffers be
	// "just pointers". If we make attribute descs also specify strides we would
	// let cookers interleave attributes as they please. The only gotcha is that
	// if we allow multiple attributes to live in the same buffer they all would
	// have to be in the same domain (per-vertex or per-triangle)

	// TODO: make this gpu-accessible
	AttributeBuffers []any

	// TODO: rename
	Parts []GeometryPart
}

// TODO: take out parameter instead of returning a new value so that the user
// can preallocate things if they want to I guess?
// TODO: make it standalone function rather than a method on Geometry?
func (m *Geometry) AccelConfig() *gpu.AccelBuildConfig {
	positions := m.AttributeBuffers[AttributePosition].(gpu.Slice[[3]float32])

	accelBuildInputs := make([]gpu.AccelBuildInput, len(m.Parts))
	for i, part := range m.Parts {
		accelBuildInputs[i] = &gpu.AccelBuildInputTriangles{
			VertexFormat:  vk.FORMAT_R32G32B32_SFLOAT,
			VertexBuffer:  gpu.UnsafePointer(gpu.SliceData(positions)),
			VertexCount:   gpu.SliceLen(positions),
			VertexStride:  int(unsafe.Sizeof(positions.Value()[0])),
			IndexType:     part.IndexBuffer.type_.vkIndexType(),
			IndexBuffer:   part.IndexBuffer.data,
			TriangleCount: part.IndexBuffer.len / 3,
		}
	}

	return &gpu.AccelBuildConfig{
		Type:   vk.ACCELERATION_STRUCTURE_TYPE_BOTTOM_LEVEL_KHR,
		Inputs: accelBuildInputs,
	}
}
