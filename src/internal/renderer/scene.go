package renderer

import (
	"structs"

	"worldspawn/geometry-go"
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

// TODO: DVec3 baseOffset

type Camera struct {
	_ structs.HostLayout

	Transform geometry.Mat4x4

	FieldOfView   float32
	NearClipPlane float32
}

type _MaterialParams struct {
	_         structs.HostLayout
	Code      gpu.Pointer[uint32]
	Triangles gpu.Pointer[[3]uint16]
	Normals   gpu.Pointer[[3]float32]
	UVs       gpu.Pointer[[2]float32]
	Hmm       [3]float32
}

// TODO: I guess we'll need to do some involved memory management in Scene.

// TODO: to abstract over still AS and motion AS we could introduce two scenes
// that implement the same interface. Though that would be insufficient of an
// abstraction as shader code would still have to be aware about still vs motion
// AS. I guess let's just not.

type light struct {
	_ structs.HostLayout

	// TODO: replace this with an instance index, geometry index and a triangle
	// triplet.
	verts [3]geometry.Vec3

	// TODO: we'll need more info for power sampling, but that seems complicated
	// in presence of materials defining emission.

	// TODO: evaluate the material for emission
	emission geometry.Vec3
}

// TODO: should probs be emissiveBLAS or something
type emissiveInstance struct {
	_ structs.HostLayout

	// TODO: we also have parts, actually!
	transform   [3][4]float32
	posBuffer   gpu.Slice[[3]float32]
	indexBuffer gpu.Slice[[3]uint16]
}

type lightSampler struct {
	_ structs.HostLayout

	// TODO: drop SamplingViewWithSampler in favor passing an image descriptor
	// and reconstructing it in the shader.
	// TODO: rename
	env gpu.SamplingViewWithSampler

	emissiveInstances     gpu.Slice[emissiveInstance]
	emissiveInstanceCount int
}

type Scene struct {
	_ structs.HostLayout

	// TODO: rename, kinda don't like what it's called
	maxPartsPerMesh int

	materialParams gpu.Slice[_MaterialParams]

	accelInstances gpu.Slice[gpu.AccelInstance]

	accel gpu.Accel

	lightSampler lightSampler

	// TODO: this probs belongs to MaterialLibrary too
	sampler gpu.Sampler

	// TODO: append experimental stuff at the end for now: we have a skill issue
	// in that we need to manually sync host and device type definitions.

	// TODO: move pipelines + SBT into a separate object that's like a
	// MaterialLibrary or whatever
	pipeline *gpu.RayTracingPipeline
	sbt      gpu.ShaderBindingTable
}

/*
type hitRecord struct {
	_      structs.HostLayout
	Handle gpu.RayTracingShaderGroupHandle
	Code   gpu.Pointer[uint32]
	// Constants gpu.UnsafePointer
	_ [64 - 32 - 8]byte // https://github.com/golang/go/issues/19057
}
*/

// TODO: we'll probs end up needing to take a struct with params
func NewScene(n int, maxPartsPerMesh int) *Scene {
	instances := gpu.MakeSliceUncached[_MaterialParams](n * maxPartsPerMesh)

	emissiveInstances := gpu.MakeSliceUncached[emissiveInstance](n)

	// TODO: make pipeline be relinked when some material is created or removed
	// or whatever.
	pipeline := gpu.LinkRayTracingShaderGroups(raygen())

	raygenRecord := gpu.NewUncached[gpu.RayTracingShaderGroupHandle]()
	*raygenRecord.Value() = raygen().Handle()

	return &Scene{
		maxPartsPerMesh: maxPartsPerMesh,
		pipeline:        pipeline, // TODO: relink this after materials change and stuff
		sbt:             gpu.MakeShaderBindingTable(raygenRecord, gpu.Slice[struct{}]{}, gpu.Slice[struct{}]{}, gpu.Slice[struct{}]{}),
		materialParams:  instances,
		accelInstances:  gpu.MakeSliceUncached[gpu.AccelInstance](n),
		accel:           gpu.NewTopLevelAccel(n),
		lightSampler: lightSampler{
			emissiveInstances: emissiveInstances,
		},
		sampler: gpu.NewSampler(&vk.SamplerCreateInfo{
			SType:            vk.STRUCTURE_TYPE_SAMPLER_CREATE_INFO,
			MinFilter:        vk.FILTER_LINEAR,
			MagFilter:        vk.FILTER_LINEAR,
			MipmapMode:       vk.SAMPLER_MIPMAP_MODE_LINEAR,
			AddressModeU:     vk.SAMPLER_ADDRESS_MODE_REPEAT,
			AddressModeV:     vk.SAMPLER_ADDRESS_MODE_REPEAT,
			AddressModeW:     vk.SAMPLER_ADDRESS_MODE_REPEAT,
			MipLodBias:       0.0, // TODO: change this every frame
			AnisotropyEnable: vk.TRUE,
			MaxAnisotropy:    8.0,
			MinLod:           0.0,
			MaxLod:           vk.LOD_CLAMP_NONE,
		}),
	}
}
