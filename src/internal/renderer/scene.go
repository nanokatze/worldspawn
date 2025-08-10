package renderer

import (
	"structs"
	"unsafe"

	"worldspawn/geometry-go"
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

// TODO: DVec3 baseOffset

const maxPartsPerMesh = 5

type Camera struct {
	Transform geometry.Mat4x4

	FieldOfView   float32
	NearClipPlane float32
}

type _MaterialParams struct {
	Triangles   gpu.Pointer[[3]uint16]
	Normals     gpu.Pointer[[3]float32]
	UVs         gpu.Pointer[[2]float32]
	TestTexture gpu.SamplingViewWithSampler
	Hmm         [3]float32
}

// TODO: I guess we'll need to do some involved memory management in Scene.

// TODO: to abstract over still AS and motion AS we could introduce two scenes
// that implement the same interface. Though that would be insufficient of an
// abstraction as shader code would still have to be aware about still vs motion
// AS. I guess let's just not.
type Scene struct {
	_ structs.HostLayout

	sky gpu.SamplingViewWithSampler

	// TODO: rename
	instances      gpu.Slice[_MaterialParams]
	accelInstances gpu.Slice[gpu.AccelInstance]
	accel          gpu.Accel

	sampler gpu.Sampler

	// TODO: append experimental stuff at the end for now: we have a skill issue
	// in that we need to manually sync host and device type definitions.

	// TODO: decouple this from instances?
	// TODO: we can make this an array of [maxPartsPerMesh] but we'll be careful
	// when specifying SBT stride
	hitRecords gpu.Slice[hitRecord]
	// TODO: delegate linked pipeline management to the user
	pipeline *gpu.RayTracingPipeline
	// TODO: construct sbt when requested rather than store?
	sbt gpu.ShaderBindingTable
}

type hitRecord struct {
	_      structs.HostLayout
	Header [32]byte
	Code   gpu.Pointer[uint32]
	_      [64 - 32 - 8]byte // https://github.com/golang/go/issues/19057
}

func NewScene(n int) *Scene {
	instances := gpu.MakeSliceUncached[_MaterialParams](n * maxPartsPerMesh)

	// TODO: pipeline linking should happen outside of the scene, actually. We
	// should just ask the user to provide us a "material library" or something.
	pipeline := gpu.LinkRayTracingShaderGroups(
		raygen(),
		sky(),
		chitInterpreter(),
	)

	raygenRecord := gpu.MakeSliceUncached[[32]byte](1)
	raygenRecord.Value()[0] = raygen().Handle()

	missRecords := gpu.MakeSliceUncached[[32]byte](1)
	missRecords.Value()[0] = sky().Handle()

	// TODO: should we decouple hit records from instance IDs?
	// TODO: with EXT_shader_invocation_reorder we could couple hit records to
	// materials instead, but we'd need an extra indirection to know which
	// material to use for each geometry (mesh part.)
	hitRecords := gpu.MakeSliceUncached[hitRecord](n * maxPartsPerMesh)

	return &Scene{
		hitRecords: hitRecords,
		pipeline:   pipeline, // TODO: relink this after materials change and stuff
		sbt: gpu.ShaderBindingTable{
			RaygenShaderRecordAddress:     gpu.UnsafePointer(gpu.SliceData(raygenRecord)),
			RaygenShaderRecordSize:        gpuSliceLenInBytes(raygenRecord),
			MissShaderBindingTableAddress: gpu.UnsafePointer(gpu.SliceData(missRecords)),
			MissShaderBindingTableSize:    gpuSliceLenInBytes(missRecords),
			MissShaderBindingTableStride:  32,
			HitShaderBindingTableAddress:  gpu.UnsafePointer(gpu.SliceData(hitRecords)),
			HitShaderBindingTableSize:     gpuSliceLenInBytes(hitRecords),
			HitShaderBindingTableStride:   int(unsafe.Sizeof(hitRecords.Value()[0])),
		},
		instances:      instances,
		accelInstances: gpu.MakeSliceUncached[gpu.AccelInstance](n),
		accel:          gpu.NewTopLevelAccel(n),
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
