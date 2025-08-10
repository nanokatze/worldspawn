package renderer

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"sync"
	"unsafe"

	"worldspawn/geometry-go"
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

type _FrameData struct {
	FrameNumber uint32

	BlueNoise gpu.SamplingView

	Proj        geometry.Mat4x4
	ProjInverse geometry.Mat4x4
	View        geometry.Mat4x4
	ViewInverse geometry.Mat4x4

	// Precomputed intermediates

	ViewProj geometry.Mat4x4 // TODO: remove?
}

var blueNoise = sync.OnceValue(func() *gpu.Image {
	gpuImg := gpu.NewImage(&gpu.ImageConfig{
		Dim:       gpu.ImageDim2D,
		Extent:    gpu.Int3{256, 256, 1},
		Layers:    8,
		MipLevels: 1,
		Samples:   1,
		Format:    vk.FORMAT_R8G8B8A8_UNORM,
		Usage:     gpu.ImageUsageSampling,
	})

	var wg gpu.WaitGroup

	// TODO: can we please not use png
	for i := range 8 {
		wg.Add(1)

		var jq gpu.JobQueue

		// TODO: come up where non-game data should live
		f, err := os.Open(fmt.Sprintf("BlueNoise/2D/256_256/LDR_RGBA_%d.png", i))
		if err != nil {
			panic(err)
		}
		defer f.Close()

		img, err := png.Decode(f)
		if err != nil {
			panic(err)
		}

		imgNRGBA := img.(*image.NRGBA)

		staging := gpu.MakeSliceUncached[byte](len(imgNRGBA.Pix))
		defer jq.Cleanup(func() { gpu.Free(gpu.UnsafePointer(gpu.SliceData(staging))) })

		copy(staging.Value(), imgNRGBA.Pix)

		gpuImg.EnqueueInit(&jq)
		gpu.EnqueueCopyMemoryToImage(
			&jq,
			gpuImg.SubImage(
				gpuImg.Dim(),
				gpuImg.Format(),
				i, i+1,
				0, 1),
			gpu.Int3{},
			staging, 0, 0,
			gpu.Int3{imgNRGBA.Rect.Max.X, imgNRGBA.Rect.Max.Y, 1})
		wg.EnqueueDone(&jq)
	}

	wg.Wait()

	return gpuImg
})

var raygen = sync.OnceValue(func() *gpu.RayTracingShaderGroup {
	return gpu.NewGeneralRayTracingShaderGroup(gpu.NewFunc(mustReadFile("shaders/scene_render.spv"), vk.SHADER_STAGE_RAYGEN_BIT_KHR, "raygen"))
})

var sky = sync.OnceValue(func() *gpu.RayTracingShaderGroup {
	sky := gpu.NewFunc(mustReadFile("shaders/scene_render.spv"), vk.SHADER_STAGE_MISS_BIT_KHR, "sky")
	return gpu.NewGeneralRayTracingShaderGroup(sky)
})

var chitInterpreter = sync.OnceValue(func() *gpu.RayTracingShaderGroup {
	chit := gpu.NewFunc(mustReadFile("shaders/scene_render.spv"), vk.SHADER_STAGE_CLOSEST_HIT_BIT_KHR, "chit")
	return gpu.NewTrianglesRayTracingShaderGroup(chit, nil)
})

func gpuSliceLenInBytes[T any](s gpu.Slice[T]) int {
	return gpu.SliceLen(s) * int(unsafe.Sizeof(*new(T)))
}

func mustReadFile(filename string) []byte {
	data, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return data
}

// +Y forward +X right +Z up to -Z forward +X right -Y up
//
// TODO: move somewhere further up
// TODO: improve naming?
var toClipSpace = geometry.Mat4x4{
	{1, 0, 0, 0},
	{0, 0, -1, 0},
	{0, -1, 0, 0},
	{0, 0, 0, 1},
}

// TODO: struct for per-view stuff, with history for TAA, etc.

// TODO: we might need more bits about the dst like format, etc, depending on if
// we do compute or what.
//
// TODO: res should probably be a rect?
//
// TODO: it would be nice if we could require that dst is either storage image
// writable, color attachment, transfer dst, ...
//
// TODO: we need to pass other stuff like max aniso etc settings...
//
// TODO: make it a method on a Scene
// func (scene *Scene) EnqueueRender(jq *gpu.JobQueue, dst *gpu.Image, res gpu.Int3, t float32, camera *Camera)
func (scene *Scene) Render(jq *gpu.JobQueue, t float32, camera *Camera, dst *gpu.Image, res gpu.Int3) {
	// TODO: we can get rid of hardware aniso sampling, which is annoying and
	// has the wrong filter for many things.
	//
	// See https://mastodon.gamedev.place/@BartWronski/112445872458391965
	//
	// TODO: samplers we need to create are really dictated by materials

	frameData := gpu.NewUncached[_FrameData]()
	defer jq.Cleanup(func() { gpu.Free(gpu.UnsafePointer(frameData)) })

	dscene := gpu.NewUncached[Scene]()
	defer jq.Cleanup(func() { gpu.Free(gpu.UnsafePointer(dscene)) })
	*dscene.Value() = *scene

	{
		proj := geometry.Mat4x4InfinitePerspective(
			float32(camera.FieldOfView),
			float32(res.X)/float32(res.Y),
			float32(camera.NearClipPlane)).
			Mul4x4(toClipSpace)

		// TODO: jitter the camera slightly using an LDS of some sort.
		//
		// See also
		// https://blog.demofox.org/2022/01/01/interleaved-gradient-noise-a-different-kind-of-low-discrepancy-sequence/

		viewInverse := camera.Transform
		view := viewInverse.Inverse()

		*frameData.Value() = _FrameData{
			FrameNumber: uint32(0),
			Proj:        proj,
			ProjInverse: proj.Inverse(),
			View:        view,
			ViewInverse: viewInverse,

			ViewProj: proj.Mul4x4(view),
		}
	}

	// RT

	{
		// TODO: arguably this is the right place to do the linking the final
		// pipeline and assemble the entire SBT. Scene should only concern
		// itself with hit groups.

		args := struct {
			Scene  gpu.Pointer[Scene]
			Camera gpu.Pointer[_FrameData]
			Out    gpu.StorageView
		}{
			Scene:  dscene,
			Camera: frameData,
			Out:    dst.LoadStoreDescriptor(),
		}
		gpu.EnqueueTraceRays(jq, scene.pipeline, scene.sbt, res.X, res.Y, 1, &args)
	}

	// TODO: compositor here, perhaps user-provided.
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
