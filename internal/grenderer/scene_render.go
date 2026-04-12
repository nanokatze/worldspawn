package grenderer

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"structs"
	"sync"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
	"worldspawn/internal/gmath"
)

type Quality struct {
	_ structs.HostLayout

	MaxBounces int32

	RussianRouletteThreshold int32
}

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

	Number uint32

	// TODO: as a temporary solution we could have a globals struct that we'd
	// pass through params, and BlueNoise among other things would live there.
	BlueNoise gpu.ImageDescriptors

	Camera cameraInternal
	Film   _Film

	Quality Quality
}

var blueNoise = sync.OnceValue(func() *gpu.Image {
	var wg gpu.WaitGroup

	jq := new(gpu.JobQueue)

	// TODO: we actually only use RG at most
	gpuImg := gpu.NewImage(
		vk.FORMAT_R16G16B16A16_UNORM,
		[]int{256, 256},
		gpu.WithLayers(8),
		gpu.WithUsage(vk.IMAGE_USAGE_SAMPLED_BIT))
	gpuImg.EnqueueInit(jq)

	// TODO: can we please not use png
	for i := range gpuImg.Layers() {
		wg.Add(1)

		jq := gpu.Fork(jq)

		func() {
			// TODO: come up where non-game data should live. Maybe embed this?
			f, err := os.Open(fmt.Sprintf("BlueNoise/2D/256_256/HDR_RGBA_%d.png", i))
			if err != nil {
				panic(err)
			}
			defer f.Close()

			img, err := png.Decode(f)
			if err != nil {
				panic(err)
			}

			imgNRGBA := img.(*image.NRGBA64)

			for i := 0; i < len(imgNRGBA.Pix); i += 2 {
				imgNRGBA.Pix[i+0], imgNRGBA.Pix[i+1] = imgNRGBA.Pix[i+1], imgNRGBA.Pix[i+0]
			}

			staging := gpu.MakeSliceUncached[byte](len(imgNRGBA.Pix))
			defer jq.Cleanup(func() { gpu.Free(gpu.UnsafePointer(gpu.SliceData(staging))) })

			copy(staging.Value(), imgNRGBA.Pix)

			gpu.EnqueueCopyMemoryToImage(
				jq,
				gpuImg.SubImage(gpu.WithLayerRange{i, i + 1}), nil,
				staging, 0, 0,
				[]int{imgNRGBA.Rect.Max.X, imgNRGBA.Rect.Max.Y})

			wg.EnqueueDone(jq)
		}()
	}

	wg.Wait()

	return gpuImg
})

var raygen = sync.OnceValue(func() *gpu.RayTracingShaderGroup {
	return gpu.NewGeneralRayTracingShaderGroup(gpu.NewRayTracingFunc(mustReadFile("shaders/grenderer_scene_render.spv"), vk.SHADER_STAGE_RAYGEN_BIT_KHR, "raygen"))
})

func mustReadFile(filename string) []byte {
	data, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return data
}

func (scene *Scene) Render(
	jq *gpu.JobQueue,
	frameNumber uint32,
	camera *Camera,
	film Film,
	quality *Quality) {
	bn := blueNoise()

	bnLayer := bn.SubImage(gpu.WithLayerRange{int(frameNumber) % bn.Layers(), int(frameNumber)%bn.Layers() + 1})
	defer jq.Cleanup(bnLayer.Destroy)

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

		viewInverse := camera.Transform
		view := viewInverse.Inverse()

		*frame.Value() = frameParams{
			Scene: dscene,

			Number: frameNumber,

			BlueNoise: bnLayer.Descriptors(),

			Camera: cameraInternal{
				Proj: proj,
				View: view,

				ViewProj:    proj.Mul(view),
				ProjInverse: proj.Inverse(),
				ViewInverse: viewInverse,
			},
			Film: _Film{
				Color:              film.Color.Descriptors(),
				DiffuseAlbedo:      film.DiffuseAlbedo.Descriptors(),
				NormalAndRoughness: film.NormalAndRoughness.Descriptors(),
				Depth:              film.Depth.Descriptors(),
				Motion:             film.Motion.Descriptors(),
			},

			Quality: *quality,
		}
	}

	gpu.EnqueueTraceRays(jq, film.Extent[:], scene.pipeline, scene.sbt, &frame)
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
