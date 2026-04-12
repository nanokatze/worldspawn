package grenderer

import (
	"structs"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
	"worldspawn/internal/gmath"
	"worldspawn/internal/grenderer/internal/material"
)

// TODO: DVec3 baseOffset

// TODO: this needs to be more abstract
// TODO: provide NewCamera with func args? Or some other kind of camera builder
// thing.
type Camera struct {
	_ structs.HostLayout

	Transform gmath.Mat4x4f32

	FieldOfView   float32
	NearClipPlane float32
}

// TODO: kill this when we make scene.Mesh and scene.MeshPart directly
// accessible. Then we can just yeet the whole thing onto gpu.
type meshPart2 struct {
	_ structs.HostLayout

	IndexBuffer IndexBuffer
	PosBuffer   gpu.Pointer[[3]float32]
	Normals     gpu.Pointer[[3]float32]
}

// TODO: kill!!!!!!!!!
type materialParams struct {
	_ structs.HostLayout

	// Doesn't even belong here
	Program material.InterpreterProgram

	MeshPart meshPart2

	// must be aligned to 8-byte boundary
	Args [256]byte
}

// TODO: I guess we'll need to do some involved memory management in Scene.

// TODO: should probs be emissiveBLAS or something
type emissiveInstance struct {
	_ structs.HostLayout

	transform             [3][4]float32
	originalInstanceIndex uint32
	originalGeometryIndex uint32 // TODO: we should not need this I think?
}

type lightAccel struct {
	_ structs.HostLayout

	envTransform gmath.Mat3x3f32
	// TODO: rename
	env gpu.ImageDescriptors

	emissiveInstances     gpu.Slice[emissiveInstance]
	emissiveInstanceCount int
}

// TODO: to abstract over still AS and motion AS we could introduce two scenes
// that implement the same interface. Though that would be insufficient of an
// abstraction as shader code would still have to be aware about still vs motion
// AS. I guess let's just not.
type Scene struct {
	_ structs.HostLayout

	maxMaterialsPerInstance int

	materialArgs gpu.Slice[materialParams]

	accelData gpu.Slice[gpu.AccelInstance]

	accel gpu.TLAS

	lightAccel lightAccel

	// TODO: this probs belongs to MaterialLibrary too
	sampler gpu.ImageSampler

	// TODO: append experimental stuff at the end for now: we have a skill issue
	// in that we need to manually sync host and device type definitions.

	// TODO: move pipelines + SBT into a separate object that's like a
	// MaterialLibrary or whatever
	pipeline *gpu.RayTracingPipeline
	sbt      gpu.ShaderBindingTable

	materialParamsHost []materialParams
	accelInstancesHost []gpu.AccelInstance
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
	materialArgs := gpu.MakeSliceUncached[materialParams](n * maxPartsPerMesh)
	clear(materialArgs.Value())

	accelData := gpu.MakeSliceUncached[gpu.AccelInstance](n)
	clear(accelData.Value())

	lightAccelData := gpu.MakeSliceUncached[emissiveInstance](n)
	clear(lightAccelData.Value())

	// TODO: make pipeline be relinked when some material is created or removed
	// or whatever.
	pipeline := gpu.LinkRayTracingShaderGroups(raygen())

	raygenRecord := gpu.NewUncached[gpu.RayTracingShaderGroupHandle]()
	*raygenRecord.Value() = raygen().Handle()

	return &Scene{
		maxMaterialsPerInstance: maxPartsPerMesh,
		pipeline:                pipeline, // TODO: relink this after materials change and stuff
		sbt:                     gpu.MakeShaderBindingTable(raygenRecord, gpu.Slice[struct{}]{}, gpu.Slice[struct{}]{}, gpu.Slice[struct{}]{}),
		materialArgs:            materialArgs,
		accelData:               accelData,
		accel:                   gpu.TLAS(gpu.NewTopLevelAccel(n)),
		lightAccel: lightAccel{
			emissiveInstances: lightAccelData,
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
		materialParamsHost: materialArgs.Value(),
		accelInstancesHost: accelData.Value(),
	}
}

// TODO: make it async, or make it device-only
func (scene *Scene) SetSky(transform gmath.Mat3x3f32, sky *gpu.Image) {
	scene.lightAccel.envTransform = transform
	scene.lightAccel.env = sky.Descriptors()
}

// TODO: this should only exist on the device/in the shader
func (scene *Scene) SetInstanceTransform(i int, x [3][4]float32) {
	scene.accelInstancesHost[i].Transform = x
}

// TODO: this should only exist on the device/in the shader
func (scene *Scene) SetInstanceGeometry(i int, mask uint8, geometry *Geometry, accel gpu.BLAS, materials []*InterpretedMaterial, materialArgs [][256]byte) {
	// TODO: this really begs for a func vararg constructor tbh.
	var accelInstance gpu.AccelInstance
	accelInstance.InstanceIDAndMask = pack24_8(0, uint32(mask))
	accelInstance.SBTOffsetAndFlags = pack24_8(uint32(i*scene.maxMaterialsPerInstance), 0)
	accelInstance.SetAccel(accel)
	scene.accelInstancesHost[i] = accelInstance

	if geometry != nil {
		if len(geometry.Parts) > scene.maxMaterialsPerInstance {
			panic("umm")
		}
		for partIdx, part := range geometry.Parts {
			scene.materialParamsHost[i*scene.maxMaterialsPerInstance+partIdx] = materialParams{
				Program: materials[partIdx].program,

				// TODO: make Mesh device-accessible so we don't have to do these redundant copies every time
				MeshPart: meshPart2{
					IndexBuffer: part.IndexBuffer,
					PosBuffer:   gpu.SliceData(geometry.AttributeBuffers[AttributePosition].(gpu.Slice[[3]float32])),
					Normals:     gpu.SliceData(geometry.AttributeBuffers[AttributeNormal].(gpu.Slice[[3]float32])),
				},

				Args: materialArgs[partIdx],
			}
		}
	}
}

func (scene *Scene) EnqueueUpdateAccel(jq *gpu.JobQueue) {
	(&gpu.AccelBuildConfig{
		Type: vk.ACCELERATION_STRUCTURE_TYPE_TOP_LEVEL_KHR,
		Inputs: []gpu.AccelBuildInput{
			&gpu.AccelBuildInputInstances{
				Instances:     gpu.SliceData(scene.accelData),
				InstanceCount: uint32(gpu.SliceLen(scene.accelData)),
			},
		},
	}).EnqueueBuild(jq, gpu.Accel(scene.accel))
}
