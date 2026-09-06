package renderer

import (
	"bytes"
	_ "embed"
	"structs"
	"sync"

	"worldspawn/gpu"
	gpuktx2 "worldspawn/gpu/image/ktx2"
	gpurt "worldspawn/gpu/raytracing"
	"worldspawn/gpu/vk"
	"worldspawn/internal/gmath"
)

//go:generate slangc -target spirv -profile spirv_1_6 -fvk-use-entrypoint-name -fvk-use-c-layout -matrix-layout-row-major -capability vk_mem_model -capability spvDescriptorHeapEXT -o _shaders.spv scene_render.slang

//go:embed _shaders.spv
var _shaders []byte

//go:embed BlueNoise/2D_256_256_HDR_RGBA.ktx2
var noiseLUT []byte

// TODO: make Scene, Camera and Film interfaces eventually?

// TODO: rename to just Camera with all fields private and provide constructors
type cameraInternal struct {
	_ structs.HostLayout

	Proj gmath.Mat4x4f32
	View gmath.Mat4x4f32

	ViewProj    gmath.Mat4x4f32 // TODO: remove?
	ProjInverse gmath.Mat4x4f32
	ViewInverse gmath.Mat4x4f32
}

// TODO: carve out some things that we should push directly and keep others behind a pointer?
// TODO: reorder the fields in here
type frameParams struct {
	_ structs.HostLayout

	Scene gpu.Pointer[Scene]

	Film _Film

	Camera cameraInternal

	Number uint32
	// TODO: as a temporary solution we could have a globals struct that we'd
	// pass through params, and BlueNoise among other things would live there.
	BlueNoise gpu.ImageDescriptor

	Quality Quality
}

var noiseImage = sync.OnceValue(func() *gpu.Image {
	img, err := gpuktx2.Decode(bytes.NewReader(noiseLUT), gpu.ImageWithUsage(vk.IMAGE_USAGE_SAMPLED_BIT))
	if err != nil {
		panic(err)
	}
	return img
})

var raygen = sync.OnceValue(func() *gpurt.RayTracingShaderGroup {
	return gpurt.NewGeneralRayTracingShaderGroup(gpurt.NewRayTracingFunc(_shaders, vk.SHADER_STAGE_RAYGEN_BIT_KHR, "raygen"))
})

func (scene *Scene) EnqueueRender(jq *gpu.JobQueue, film Film, camera *Camera, cameraTransform gmath.Mat4x4f32, frameNumber uint32, quality *Quality) {
	noise := noiseImage()

	noiseLayer := noise.SubImage(gpu.SliceLayers{int(frameNumber) % noise.Layers(), int(frameNumber)%noise.Layers() + 1})
	defer jq.Cleanup(noiseLayer.Destroy)

	dscene := gpu.NewUncached[Scene]()
	*dscene.Value() = *scene
	defer jq.Cleanup(func() { gpu.Free(gpu.UnsafePointer(dscene)) })

	frame := gpu.NewUncached[frameParams]()
	defer jq.Cleanup(func() { gpu.Free(gpu.UnsafePointer(frame)) })

	{
		proj := gmath.Mat4x4InfinitePerspective(
			float32(camera.FieldOfView),
			float32(film.Extent[0])/float32(film.Extent[1]), // TODO: add a field to Camera instead?
			float32(camera.NearClipPlane))

		// TODO: jitter the camera slightly using an LDS of some sort.
		// TODO: DLSS strongly recommends using halton as it's trained on it or
		// whatever.
		//
		// See also
		// https://blog.demofox.org/2022/01/01/interleaved-gradient-noise-a-different-kind-of-low-discrepancy-sequence/

		viewInverse := cameraTransform
		view := viewInverse.Inv()

		*frame.Value() = frameParams{
			Film: _Film{
				Color:              film.Color.Descriptor(),
				DiffuseAlbedo:      film.DiffuseAlbedo.Descriptor(),
				NormalAndRoughness: film.NormalAndRoughness.Descriptor(),
				Depth:              film.Depth.Descriptor(),
				Motion:             film.Motion.Descriptor(),
			},

			Scene: dscene,

			Camera: cameraInternal{
				Proj: proj,
				View: view,

				ViewProj:    proj.Mul(view),
				ProjInverse: proj.Inv(),
				ViewInverse: viewInverse,
			},

			Number: frameNumber,

			BlueNoise: noiseLayer.Descriptor(),

			Quality: *quality,
		}
	}

	gpurt.EnqueueTraceRays(jq, film.Extent[:], scene.pipeline, scene.sbt, &frame)
}

// TODO: move into gpu/vk or at least just gpu?
func pack24_8(x, y uint32) uint32 {
	if x >= 1<<24 {
		panic("wtf")
	}
	if y >= 1<<8 {
		panic("wtf")
	}
	return x | y<<24
}
