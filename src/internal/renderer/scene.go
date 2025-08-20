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

// TODO: make NewMaterial etc update these
var rtCallables gpu.Slice[struct{}]
var rtPipeline *gpu.RayTracingPipeline

// TODO: I guess we'll need to do some involved memory management in Scene.

// TODO: to abstract over still AS and motion AS we could introduce two scenes
// that implement the same interface. Though that would be insufficient of an
// abstraction as shader code would still have to be aware about still vs motion
// AS. I guess let's just not.
type Scene struct {
	_ structs.HostLayout

	// TODO: rename, kinda don't like what it's called
	maxPartsPerMesh int

	sky gpu.SamplingViewWithSampler

	// TODO: rename
	instances      gpu.Slice[_MaterialParams]
	accelInstances gpu.Slice[gpu.AccelInstance]
	accel          gpu.Accel

	sampler gpu.Sampler

	// TODO: append experimental stuff at the end for now: we have a skill issue
	// in that we need to manually sync host and device type definitions.

	// TODO: delegate linked pipeline management to the user
	// TODO: actually, have global pipelines for these
	pipeline *gpu.RayTracingPipeline
	// TODO: construct sbt when requested rather than store?
	sbt gpu.ShaderBindingTable
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

	// TODO: make pipeline be relinked when some material is created or removed
	// or whatever.
	pipeline := gpu.LinkRayTracingShaderGroups(raygen())

	raygenRecord := gpu.NewUncached[gpu.RayTracingShaderGroupHandle]()
	*raygenRecord.Value() = raygen().Handle()

	return &Scene{
		maxPartsPerMesh: maxPartsPerMesh,
		pipeline:        pipeline, // TODO: relink this after materials change and stuff
		sbt:             gpu.MakeShaderBindingTable(raygenRecord, gpu.Slice[struct{}]{}, gpu.Slice[struct{}]{}, gpu.Slice[struct{}]{}),
		instances:       instances,
		accelInstances:  gpu.MakeSliceUncached[gpu.AccelInstance](n),
		accel:           gpu.NewTopLevelAccel(n),
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
