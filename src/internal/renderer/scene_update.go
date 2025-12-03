package renderer

import (
	"worldspawn/geometry-go"
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

type Instance struct {
	Transform int

	Mask uint8
}

// TODO: we'll also want to specify the parameters
// TODO: move Material into its own column (array)?
type MaterialInstance struct {
	Material *InterpretedMaterial
	Args     [256]byte
}

// TODO: actually remove this entirely from here and push any kind of tracking
// onto the user.
type SceneDirty struct {
	Sky *gpu.Image

	// TODO: remove this field
	OurCamera Camera

	Parent      []int
	TransformT0 []geometry.TRS3
	TransformT1 []geometry.TRS3
	// TODO: we also need to carry velocity here for motion blur, or at least
	// some extra info to disambiguate fast temporally-aliased motions.

	// TODO: prefix instance-rate stuff with instance? or idk Or put them into
	// Instance struct { ... }
	Instance  []Instance
	Mesh      []*Mesh
	Materials [][]MaterialInstance
}

func NewSceneDirty(n int) *SceneDirty {
	return &SceneDirty{
		Parent:      make([]int, n),
		TransformT0: make([]geometry.TRS3, n),
		TransformT1: make([]geometry.TRS3, n),

		Instance:  make([]Instance, n),
		Mesh:      make([]*Mesh, n),
		Materials: make([][]MaterialInstance, n),
	}
}

// TODO: rename to something like GlobalTransform?
func (s *SceneDirty) Transform(i int, t float32) geometry.Mat4x4 {
	B := geometry.Mat4x4Identity()
	for ; i != 0; i = s.Parent[i] {
		A := s.TransformT0[i].NLerp(s.TransformT1[i], t).Mat4x4()
		B = A.Mul4x4(B)
	}
	return B
}

// Not sure whether this should take t. We might wanna make it a separate
// method.
// TODO: do not do any host state management here but delegate things to the
// user where applicable. E.g. for materials we'll want the user to be p
// explicit about them (assign materials indices or w/e)
func (scene *Scene) EnqueueUpdate(jq *gpu.JobQueue, dirty *SceneDirty, t float32) {
	// TODO: update sky device-side too? That would make things extra neat and
	// fun.
	// TODO: hard-require sky?
	if dirty.Sky != nil {
		scene.lightSampler.env = dirty.Sky.SamplingDescriptor().WithSampler(scene.sampler)
	} else {
		scene.lightSampler.env = gpu.SamplingViewWithSampler{}
	}

	// BUG: writes should happen after all commands in jq complete. We should
	// just put things into a staging buffer and perform a copy on the device.

	var instanceCount int
	var emissiveInstanceCount int

	materialParamsHost := scene.materialParams.Value()
	accelInstancesHost := scene.accelInstances.Value()
	emissiveInstancesHost := scene.lightSampler.emissiveInstances.Value()
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

		for partIdx, part := range mesh.parts {
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
			if materialInstance.Material.emissive() {
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
			accel = mesh.Accel.Data
		}

		accelInstancesHost[instanceIdx] = gpu.AccelInstance{
			Transform:         A,
			InstanceIDAndMask: pack24_8(0, uint32(instance.Mask)),
			SBTOffsetAndFlags: pack24_8(uint32(instanceIdx*scene.maxPartsPerMesh), 0),
			Accel:             accel,
		}

		instanceCount = max(instanceCount, instanceIdx+1)
	}

	scene.lightSampler.emissiveInstanceCount = emissiveInstanceCount

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
