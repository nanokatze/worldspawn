package main

import (
	"slices"

	"worldspawn/gpu"
	"worldspawn/internal/animgraph"
	"worldspawn/internal/geometry"
	"worldspawn/internal/gmath"
	"worldspawn/internal/renderer"
)

type gsdata struct {
	// scratch
	//
	// TODO: split scratch into host->device scratch and on-device scratch. The
	// latter could feasibly be served from a fixed size allocation.

	pose             gpu.Slice[gmath.Mat4x4f32]
	skinnedPositions gpu.Slice[[3]float32]
	skinnedNormals   gpu.Slice[[3]float32] // not used right now

	// output

	geometry *renderer.Geometry
	accel    gpu.BLAS
}

// TODO: rename this
type geoNodes struct {
	src    *fileBackedMesh
	skelly *animgraph.Skeleton
	pose   animgraph.Pose
}

// TODO: factor out allocation requests into its own function so that we can run
// that once every update. It would also probably be nice to distinguish GSes
// that need to be run every time we get a snapshot, or every frame.

// TODO: rename?

func (gs *geoNodes) Outputs(data *gsdata) (*renderer.Geometry, gpu.BLAS) {
	if len(gs.pose.Bones) == 0 {
		if gs.src == nil {
			return nil, gpu.BLAS{}
		}
		return &gs.src.geometry, gs.src.accel
	}
	return data.geometry, data.accel
}

func (gs *geoNodes) EnqueueEvaluate(jq *gpu.JobQueue, data *gsdata) {
	if len(gs.pose.Bones) == 0 {
		return
	}

	rest := gs.src.geometry

	restPositions := gs.src.attrs[renderer.AttributePosition].(gpu.Slice[[3]float32])

	skinnedPositions := data.skinnedPositions
	if gpu.SliceCap(skinnedPositions) < gpu.SliceLen(restPositions) {
		skinnedPositions = gpu.MakeSliceUncached[[3]float32](gpu.SliceLen(restPositions))
	}
	skinnedPositions = skinnedPositions.Slice(0, gpu.SliceLen(restPositions))

	data.skinnedPositions = skinnedPositions

	// TODO: we should probably have our own geometry structure with a method to
	// copy it over into pathtracer.Geometry so that we don't need to do this
	// weird patching.
	skinned := new(renderer.Geometry)
	skinned.AttributeBuffers = slices.Clone(rest.AttributeBuffers)
	skinned.AttributeBuffers[renderer.AttributePosition] = skinnedPositions
	skinned.Parts = slices.Clone(gs.src.geometry.Parts)
	data.geometry = skinned

	if gpu.SliceLen(data.pose) < len(gs.src.joints) {
		println("allocating pose", len(gs.src.joints))
		data.pose = gpu.MakeSliceUncached[gmath.Mat4x4f32](len(gs.src.joints))
	}

	poseHost := data.pose.Value()
	for i, name := range gs.src.joints {
		m, ok := gs.pose.Bones[gs.skelly.JointByName(name)]
		if ok {
			poseHost[i] = m.ToMat()
		} else {
			poseHost[i] = gmath.Mat4x4One[float32]()
		}
	}

	accelConfig := skinned.AccelConfig()
	accelSizes := accelConfig.CalcSizes()
	if data.accel.Size() < accelSizes.Accel {
		println("allocating accel", accelSizes.Accel)
		data.accel = gpu.NewBLAS(accelSizes.Accel)
	}

	// TODO: we need to run this every frame, interpolating stuff.
	geometry.EnqueueSkinMesh(jq, skinnedPositions, restPositions, gs.src.jointWeights, gs.src.jointsPerVertex, data.pose)

	data.accel.EnqueueBuild(jq, accelConfig)
}
