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

// TODO: to abstract over still AS and motion AS we could introduce two scenes
// that implement the same interface. Though that would be insufficient of an
// abstraction as shader code would still have to be aware about still vs motion
// AS. I guess let's just not.
type Scene struct {
	sky gpu.SamplingView

	tlasInstances gpu.Slice[gpu.AccelInstance]
	tlas          gpu.UnsafePointer // TODO: maybe make this a proper typedef
}

func NewScene(n int) *Scene {
	tlasInstances := gpu.MakeSliceUncached[gpu.AccelInstance](n)

	tlasConfig := &gpu.AccelBuildConfig{
		Type: vk.ACCELERATION_STRUCTURE_TYPE_TOP_LEVEL_KHR,
		Inputs: []gpu.AccelBuildInput{
			&gpu.AccelBuildInputInstances{
				InstanceCount: uint32(n),
			},
		},
	}
	tlasSize, _, _ := tlasConfig.CalcSizes()
	tlas := gpu.UnsafePointer(gpu.SliceData(gpu.MakeSliceUncached[byte](tlasSize)))

	return &Scene{
		tlasInstances: tlasInstances,
		tlas:          tlas,
	}
}

type Instance struct {
	Transform int

	Mask uint8 // should this be moved into MeshInstance?

	// BLAS gpu.UnsafePointer
}

type MeshInstance struct {
	Mesh *Mesh

	BaseMesh *Mesh
}

// TODO: make all of the fields private and provide methods for manipulation.
// This is necessary for dirty trackers, which will in turn allow us to
// implement fine-grained update.
type SceneDirty struct {
	Sky gpu.SamplingView

	// TODO: remove this field
	OurCamera Camera

	Parent      []int
	TransformT0 []geometry.TRS3
	TransformT1 []geometry.TRS3
	// TODO: we also need to carry velocity here for motion blur, or at least
	// some extra info to disambiguate fast temporally-aliased motions.

	// TODO: should instances and cameras live with their own indexing, separate
	// from transforms? It could result in both additional complexity in some
	// areas and simplifications in others

	Instance     []Instance
	MeshInstance []MeshInstance
	// TODO: while Instance should contain common instance things
	// like Transform and BLAS, instance-type specific information should
	// go into its own array, e.g. meshes go into MeshInstance and e.g.
	// VoxelInstance goes into its own.

	// This is a hack for testing skeletal posing. This should only live in the
	// user's scene type and not here.
	Pose [][]geometry.Mat4x4
}

func NewSceneDirty(n int) *SceneDirty {
	return &SceneDirty{
		Parent:      make([]int, n),
		TransformT0: make([]geometry.TRS3, n),
		TransformT1: make([]geometry.TRS3, n),

		Instance:     make([]Instance, n),
		MeshInstance: make([]MeshInstance, n),

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
	scene.sky = dirty.Sky

	// BUG: writes should happen after all commands in jq complete. We should
	// just put things into a staging buffer and perform a copy on the device.

	var instanceCount int

	tlasInstancesHost := scene.tlasInstances.Value()
	for instanceIndex, instance := range dirty.Instance {
		// TODO: is this necessary?
		if instanceIndex == 0 || instance.Transform == 0 {
			tlasInstancesHost[instanceIndex] = gpu.AccelInstance{}
			continue
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

		tlasInstancesHost[instanceIndex].Transform = A
		tlasInstancesHost[instanceIndex].InstanceIDAndMask = pack24_8(uint32(instanceIndex), uint32(instance.Mask))
		tlasInstancesHost[instanceIndex].SBTOffsetAndFlags = pack24_8(0, 0)

		meshInstance := dirty.MeshInstance[instanceIndex]

		// TODO: is this necessary?
		var blas gpu.UnsafePointer
		if meshInstance.Mesh != nil {
			blas = meshInstance.Mesh.BLAS
		}
		tlasInstancesHost[instanceIndex].Accel = blas

		instanceCount = max(instanceCount, instanceIndex+1)
	}

	tlasConfig := &gpu.AccelBuildConfig{
		Type: vk.ACCELERATION_STRUCTURE_TYPE_TOP_LEVEL_KHR,
		Inputs: []gpu.AccelBuildInput{
			&gpu.AccelBuildInputInstances{
				Instances:     gpu.SliceData(scene.tlasInstances),
				InstanceCount: uint32(instanceCount),
			},
		},
	}

	accelerationStructureSize, buildScratchSize, _ := tlasConfig.CalcSizes()

	tlasBuildScratch := gpu.UnsafePointer(gpu.SliceData(
		gpu.MakeSliceUncached[byte](buildScratchSize)))
	defer jq.Cleanup(func() { gpu.Free(tlasBuildScratch) })
	gpu.EnqueueAccelBuild(jq,
		scene.tlas,
		accelerationStructureSize,
		tlasConfig,
		tlasBuildScratch)
}
