package pathtracer

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"sync"

	"worldspawn/geometry-go"
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

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

type Quality struct {
	MaxBounces int32

	RussianRouletteThreshold int32
}

type _FrameData struct {
	Counter uint32 // TODO: rename and give it a better name

	// TODO: would be nice to sneak it into a global thing somewhere somehow. It
	// doesn't really belong here and we could just choose the layer in the
	// shader after all tbh.
	BlueNoise gpu.SamplingView

	// Outline these?
	Proj geometry.Mat4x4
	View geometry.Mat4x4

	Quality Quality

	// Precomputed

	ViewProj    geometry.Mat4x4 // TODO: remove?
	ProjInverse geometry.Mat4x4
	ViewInverse geometry.Mat4x4
}

var blueNoise = sync.OnceValue(func() *gpu.Image {
	var wg gpu.WaitGroup

	var jq gpu.JobQueue

	gpuImg := gpu.NewImage(&gpu.ImageConfig{
		Dim:       gpu.ImageDim2D,
		Extent:    [3]int{256, 256, 1},
		Layers:    8,
		MipLevels: 1,
		Samples:   1,
		Format:    vk.FORMAT_R16G16B16A16_UNORM, // TODO: we actually only use RG at most
		Usage:     gpu.ImageUsageSampling,
	})
	gpuImg.EnqueueInit(&jq)

	// TODO: can we please not use png
	for i := range gpuImg.Layers() {
		wg.Add(1)

		jq := jq.Fork()

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
				gpuImg.SubImage(
					gpuImg.Dim(),
					gpuImg.Format(),
					i, i+1,
					0, 1),
				[3]int{},
				staging, 0, 0,
				[3]int{imgNRGBA.Rect.Max.X, imgNRGBA.Rect.Max.Y, 1})

			wg.EnqueueDone(jq)
		}()
	}

	wg.Wait()

	return gpuImg
})

var raygen = sync.OnceValue(func() *gpu.RayTracingShaderGroup {
	return gpu.NewGeneralRayTracingShaderGroup(gpu.NewFunc(mustReadFile("shaders/scene_render.spv"), vk.SHADER_STAGE_RAYGEN_BIT_KHR, "raygen"))
})

func mustReadFile(filename string) []byte {
	data, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return data
}

// TODO: change fn to be an int?
func (scene *Scene) Render(
	jq *gpu.JobQueue,
	film Film,
	fn uint32,
	camera *Camera,
	quality *Quality) {
	if film.Extent[2] != 1 {
		panic("film.Extent[2] must be 1")
	}

	bn := blueNoise()

	bnLayer := bn.SubImage(
		bn.Dim(),
		bn.Format(),
		int(fn)%bn.Layers(), int(fn)%bn.Layers()+1,
		0, 1)
	defer jq.Cleanup(bnLayer.Destroy)

	frameData := gpu.NewUncached[_FrameData]()
	defer jq.Cleanup(func() { gpu.Free(gpu.UnsafePointer(frameData)) })

	{
		proj := geometry.Mat4x4InfinitePerspective(
			float32(camera.FieldOfView),
			float32(film.Extent[0])/float32(film.Extent[1]), // TODO: add a field to Camera instead?
			float32(camera.NearClipPlane)).
			Mul4x4(toClipSpace)

		// TODO: jitter the camera slightly using an LDS of some sort.
		// TODO: DLSS strongly recommends using halton as it's trained on it or
		// whatever.
		//
		// See also
		// https://blog.demofox.org/2022/01/01/interleaved-gradient-noise-a-different-kind-of-low-discrepancy-sequence/

		viewInverse := camera.Transform
		view := viewInverse.Inverse()

		*frameData.Value() = _FrameData{
			Counter: fn,

			BlueNoise: bnLayer.SamplingDescriptor(),

			Proj: proj,
			View: view,

			Quality: *quality,
			// MaxBounces: 2,

			// RussianRouletteThreshold: 1,

			ViewProj:    proj.Mul4x4(view),
			ProjInverse: proj.Inverse(),
			ViewInverse: viewInverse,
		}
	}

	dscene := gpu.NewUncached[Scene]()
	*dscene.Value() = *scene
	defer jq.Cleanup(func() { gpu.Free(gpu.UnsafePointer(dscene)) })

	args := struct {
		Scene  gpu.Pointer[Scene]
		Camera gpu.Pointer[_FrameData]
		Out    gpu.StorageView
	}{
		Scene:  dscene,
		Camera: frameData,
		Out:    film.Color.LoadStoreDescriptor(),
	}
	gpu.EnqueueTraceRays(jq, film.Extent, scene.pipeline, scene.sbt, &args)
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
