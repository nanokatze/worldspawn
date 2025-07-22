package renderer

import (
	"worldspawn/geometry-go"
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

// TODO: DVec3 baseOffset

type Camera struct {
	Transform geometry.Mat4x4

	FieldOfView   float32
	NearClipPlane float32
}

type _Mesh struct {
	Primitives  gpu.Pointer[[3]uint16]
	UVs         gpu.Pointer[[2]float32]
	TestTexture gpu.SamplingViewWithSampler
}

// TODO: I guess we'll need to do some involved memory management in Scene.

// TODO: to abstract over still AS and motion AS we could introduce two scenes
// that implement the same interface. Though that would be insufficient of an
// abstraction as shader code would still have to be aware about still vs motion
// AS. I guess let's just not.
//
// TODO: Scene needs to also manage the SBT
type Scene struct {
	sky gpu.SamplingViewWithSampler

	instances      gpu.Slice[gpu.Pointer[_Mesh]]
	accelInstances gpu.Slice[gpu.AccelInstance]
	accel          gpu.Accel

	// TODO: remove once we move to stochastic sampling
	sampler gpu.Sampler
}

func NewScene(n int) *Scene {
	instances := gpu.MakeSliceUncached[gpu.Pointer[_Mesh]](n)
	hack := gpu.MakeSliceUncached[_Mesh](n)
	instancesHost := instances.Value()
	for i := range instancesHost {
		instancesHost[i] = hack.Index(i)
	}

	return &Scene{
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

type Instance struct {
	Transform int

	Mask uint8
}

// TODO: we'll also want to specify the parameters
// TODO: rename to MaterialInstance?
type Material struct {
	// TODO: this should be in a separate array, this specifies a material
	TestTexture *gpu.Image
}

// TODO: make all of the fields private and provide methods for manipulation.
// This is necessary for dirty trackers, which will in turn allow us to
// implement fine-grained update.
// TODO: rename to something like SceneUpdateBatch or idk?
type SceneDirty struct {
	Sky *gpu.Image

	// TODO: remove this field
	OurCamera Camera

	Parent      []int
	TransformT0 []geometry.TRS3
	TransformT1 []geometry.TRS3
	// TODO: we also need to carry velocity here for motion blur, or at least
	// some extra info to disambiguate fast temporally-aliased motions.

	Instance []Instance
	Mesh     []*Mesh
	// Vox []*Vox
	Materials [][]Material

	// TODO: this is a hack for testing skeletal posing and should be removed.
	Pose [][]geometry.Mat4x4
}

func NewSceneDirty(n int) *SceneDirty {
	return &SceneDirty{
		Parent:      make([]int, n),
		TransformT0: make([]geometry.TRS3, n),
		TransformT1: make([]geometry.TRS3, n),

		Instance:  make([]Instance, n),
		Mesh:      make([]*Mesh, n),
		Materials: make([][]Material, n),

		Pose: make([][]geometry.Mat4x4, n),
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
func (scene *Scene) EnqueueUpdate(jq *gpu.JobQueue, dirty *SceneDirty, t float32) {
	// TODO: update sky device-side too? That would make things extra neat and
	// fun.
	// TODO: hard-require sky?
	if dirty.Sky != nil {
		scene.sky = dirty.Sky.SamplingDescriptor().WithSampler(scene.sampler)
	} else {
		scene.sky = gpu.SamplingViewWithSampler{}
	}

	// BUG: writes should happen after all commands in jq complete. We should
	// just put things into a staging buffer and perform a copy on the device.

	var instanceCount int

	instancesHost := scene.instances.Value()
	accelInstancesHost := scene.accelInstances.Value()
	for instanceIndex, instance := range dirty.Instance {
		// TODO: is this necessary?
		if instanceIndex == 0 || instance.Transform == 0 {
			accelInstancesHost[instanceIndex] = gpu.AccelInstance{}
			continue
		}

		mesh := dirty.Mesh[instanceIndex]

		*instancesHost[instanceIndex].Value() = _Mesh{
			Primitives:  gpu.SliceData(mesh.primitives),
			UVs:         gpu.SliceData(mesh.uvs),
			TestTexture: dirty.Materials[instanceIndex][0].TestTexture.SamplingDescriptor().WithSampler(scene.sampler),
		}

		// Should be done on the device
		instanceTransform := dirty.Transform(instanceIndex, t)

		// TODO: outline into a func
		var A [3][4]float32
		for i := range A {
			for j := range A[i] {
				A[i][j] = instanceTransform[i][j]
			}
		}

		accelInstancesHost[instanceIndex].Transform = A
		accelInstancesHost[instanceIndex].InstanceIDAndMask = pack24_8(uint32(instanceIndex), uint32(instance.Mask))
		accelInstancesHost[instanceIndex].SBTOffsetAndFlags = pack24_8(0, 0)

		// TODO: is this necessary?
		var accel gpu.UnsafePointer
		if mesh != nil {
			accel = mesh.Accel.Data
		}
		accelInstancesHost[instanceIndex].Accel = accel

		instanceCount = max(instanceCount, instanceIndex+1)
	}

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
