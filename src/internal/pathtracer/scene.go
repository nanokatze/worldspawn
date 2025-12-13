package pathtracer

import (
	"structs"

	"worldspawn/geometry-go"
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
	"worldspawn/internal/pathtracer/internal/material"
)

// TODO: DVec3 baseOffset

// TODO: this needs to be more abstract
type Camera struct {
	_ structs.HostLayout

	Transform geometry.Mat4x4

	FieldOfView   float32
	NearClipPlane float32
}

// TODO: kill this when we make scene.Mesh and scene.MeshPart directly
// accessible. Then we can just yeet the whole thing onto gpu.
type meshPart2 struct {
	_ structs.HostLayout

	Triangles    gpu.Pointer[[3]uint16]
	NumTriangles uint32
	PosBuffer    gpu.Pointer[[3]float32]
	Normals      gpu.Pointer[[3]float32]
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

	// TODO: drop SamplingViewWithSampler in favor passing an image descriptor
	// and reconstructing it in the shader.
	// TODO: rename
	env gpu.SamplingViewWithSampler

	emissiveInstances     gpu.Slice[emissiveInstance]
	emissiveInstanceCount int
}

// TODO: to abstract over still AS and motion AS we could introduce two scenes
// that implement the same interface. Though that would be insufficient of an
// abstraction as shader code would still have to be aware about still vs motion
// AS. I guess let's just not.
type Scene struct {
	_ structs.HostLayout

	// TODO: rename, kinda don't like what it's called
	maxPartsPerMesh int

	// TODO: rename to materialData? materialInstanceData?
	// materialInstanceParams?
	materialParams gpu.Slice[materialParams]

	// TODO: rename to accelData etc?
	accelInstances gpu.Slice[gpu.AccelInstance]

	accel gpu.Accel

	lightAccel lightAccel

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
	instances := gpu.MakeSliceUncached[materialParams](n * maxPartsPerMesh)

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
		lightAccel: lightAccel{
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

// TODO: do not do any host management in EnqueueUpdate (obviously.) Accept a
// list of transforms and instances to update.
func (scene *Scene) EnqueueUpdate(jq *gpu.JobQueue, dirty *SceneUpdate, t float32) {
	// TODO: update sky device-side too? That would make things extra neat and
	// fun.
	// TODO: hard-require sky?
	if dirty.Sky != nil {
		scene.lightAccel.env = dirty.Sky.SamplingDescriptor().WithSampler(scene.sampler)
	} else {
		scene.lightAccel.env = gpu.SamplingViewWithSampler{}
	}

	// BUG: writes should happen after all commands in jq complete. We should
	// just put things into a staging buffer and perform a copy on the device.

	var instanceCount int
	var emissiveInstanceCount int

	materialParamsHost := scene.materialParams.Value()
	accelInstancesHost := scene.accelInstances.Value()
	emissiveInstancesHost := scene.lightAccel.emissiveInstances.Value()
	for instanceIdx, instance := range dirty.Instance {
		// TODO: is this necessary?
		// TODO: actually we might have instances at index 0: this would happen
		// if the entity at index 0 is in the next generation. Maybe we should
		// just forbid index 0 in our entities.
		if instanceIdx == 0 || instance.Transform == 0 {
			accelInstancesHost[instanceIdx] = gpu.AccelInstance{}
			continue
		}

		// Should be done on the device
		instanceTransform := dirty.Transform(instanceIdx, t)

		// TODO: outline into a func
		var A [3][4]float32
		for i := range A {
			for j := range A[i] {
				A[i][j] = instanceTransform[i][j]
			}
		}

		mesh := dirty.Mesh[instanceIdx]

		for partIdx, part := range mesh.Parts {
			materialInstance := dirty.Materials[instanceIdx][partIdx]

			materialParamsHost[instanceIdx*scene.maxPartsPerMesh+partIdx] = materialParams{
				Program: materialInstance.Material.program,

				// TODO: make Mesh device-accessible so we don't have to do these redundant copies every time
				MeshPart: meshPart2{
					Triangles:    gpu.SliceData(part.IndexBuffer),
					NumTriangles: uint32(gpu.SliceLen(part.IndexBuffer)),
					PosBuffer:    gpu.SliceData(part.PosBuffer),
					Normals:      gpu.SliceData(part.NormalBuffer),
				},

				Args: materialInstance.Args,
			}

			// TODO: we should build an emissive blas and when instancing it
			// we'll enable/disable geometries (by specifying emission power for
			// those geometries)
			if materialInstance.Material.emissive {
				emissiveInstancesHost[emissiveInstanceCount] = emissiveInstance{
					transform:             A,
					originalInstanceIndex: uint32(instanceIdx),
					originalGeometryIndex: uint32(partIdx),
				}
				emissiveInstanceCount++
			}
		}

		// TODO: is this necessary?
		var accel gpu.UnsafePointer
		if mesh != nil {
			accel = mesh.accel.Data
		}

		accelInstancesHost[instanceIdx] = gpu.AccelInstance{
			Transform:         A,
			InstanceIDAndMask: pack24_8(0, uint32(instance.Mask)),
			SBTOffsetAndFlags: pack24_8(uint32(instanceIdx*scene.maxPartsPerMesh), 0),
			Accel:             accel,
		}

		instanceCount = max(instanceCount, instanceIdx+1)
	}

	scene.lightAccel.emissiveInstanceCount = emissiveInstanceCount

	scene.accel.EnqueueBuild(jq,
		&gpu.AccelBuildConfig{
			Type: vk.ACCELERATION_STRUCTURE_TYPE_TOP_LEVEL_KHR,
			Inputs: []gpu.AccelBuildInput{
				&gpu.AccelBuildInputInstances{
					Instances:     gpu.SliceData(scene.accelInstances),
					InstanceCount: uint32(instanceCount),
				},
			},
		})
}
