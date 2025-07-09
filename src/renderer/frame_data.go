package renderer

import (
	"worldspawn/geometry-go"
	"worldspawn/gpu"
)

type _FrameData struct {
	FrameNumber uint32

	BlueNoise gpu.SamplingView

	Sky gpu.SamplingViewWithSampler

	Proj        geometry.Mat4x4
	ProjInverse geometry.Mat4x4
	View        geometry.Mat4x4
	ViewInverse geometry.Mat4x4

	// Precomputed intermediates

	ViewProj geometry.Mat4x4 // TODO: remove?
}
