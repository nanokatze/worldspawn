package renderer

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

var pathTracer = sync.OnceValues(func() (*gpu.RayTracingPipeline, gpu.ShaderBindingTable) {
	raygen := gpu.NewFunc(mustReadFile("shaders/rt.spv"), vk.SHADER_STAGE_RAYGEN_BIT_KHR, "raygen")
	raygenSG := gpu.NewRayTracingShaderGroup(vk.RAY_TRACING_SHADER_GROUP_TYPE_GENERAL_KHR, raygen, nil, nil, nil)

	sky := gpu.NewFunc(mustReadFile("shaders/rt.spv"), vk.SHADER_STAGE_MISS_BIT_KHR, "sky")
	skySG := gpu.NewRayTracingShaderGroup(vk.RAY_TRACING_SHADER_GROUP_TYPE_GENERAL_KHR, sky, nil, nil, nil)

	chit := gpu.NewFunc(mustReadFile("shaders/rt.spv"), vk.SHADER_STAGE_CLOSEST_HIT_BIT_KHR, "chit")
	chitSG := gpu.NewRayTracingShaderGroup(vk.RAY_TRACING_SHADER_GROUP_TYPE_TRIANGLES_HIT_GROUP_KHR, nil, chit, nil, nil)

	chit2 := gpu.NewFunc(mustReadFile("shaders/rt.spv"), vk.SHADER_STAGE_CLOSEST_HIT_BIT_KHR, "chit2")
	chit2SG := gpu.NewRayTracingShaderGroup(vk.RAY_TRACING_SHADER_GROUP_TYPE_TRIANGLES_HIT_GROUP_KHR, nil, chit2, nil, nil)

	linked := gpu.LinkRTStuffTogether([]*gpu.RayTracingShaderGroup{
		// These can appear in any order
		raygenSG,
		skySG,
		chitSG,
		chit2SG,
	})

	// TODO: make shaderBindingTableBuilder

	raygenRecord := gpu.MakeSliceUncached[byte](32)
	copy(raygenRecord.Value(), raygenSG.Handle())

	missRecords := gpu.MakeSliceUncached[byte](1 * 32)
	copy(missRecords.Value(), skySG.Handle())

	hitRecords := gpu.MakeSliceUncached[byte](2 * 32)
	copy(hitRecords.Value()[0*32:], chitSG.Handle())
	copy(hitRecords.Value()[1*32:], chit2SG.Handle())

	return linked, gpu.ShaderBindingTable{
		RaygenShaderRecordAddress:     gpu.UnsafePointer(gpu.SliceData(raygenRecord)),
		RaygenShaderRecordSize:        gpu.SliceLen(raygenRecord),
		MissShaderBindingTableAddress: gpu.UnsafePointer(gpu.SliceData(missRecords)),
		MissShaderBindingTableSize:    gpu.SliceLen(missRecords),
		MissShaderBindingTableStride:  32,
		HitShaderBindingTableAddress:  gpu.UnsafePointer(gpu.SliceData(hitRecords)),
		HitShaderBindingTableSize:     gpu.SliceLen(hitRecords),
		HitShaderBindingTableStride:   32,
	}
})

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

// TODO: rename this to e.g. History, make all fields private and make
// Renderer.Render a standalone function.
type Renderer struct {
	// TODO: other things here like proj and view from the last frame, TAA image history, etc.

	ctr uint32
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
func (re *Renderer) Render(jq *gpu.JobQueue, scene *Scene, t float32, camera *Camera, dst *gpu.Image, res gpu.Int3, testTexture *Texture) {
	// TODO: we can get rid of hardware aniso sampling, which is annoying and
	// has the wrong filter for many things.
	//
	// See https://mastodon.gamedev.place/@BartWronski/112445872458391965
	//
	// TODO: samplers we need to create are really dictated by materials
	sampler := gpu.NewSampler(&vk.SamplerCreateInfo{
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
	})
	defer jq.Cleanup(sampler.Destroy)

	blueNoise := blueNoise()
	blueNoiseView := blueNoise.SubImage(
		blueNoise.Dim(),
		blueNoise.Format(),
		int(re.ctr%8), int(re.ctr%8)+1,
		0, 1).
		NewSamplingView()
	defer jq.Cleanup(blueNoiseView.Destroy)

	frameData := gpu.NewUncached[_FrameData]()
	defer jq.Cleanup(func() { gpu.Free(gpu.UnsafePointer(frameData)) })

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
			FrameNumber: uint32(re.ctr),
			BlueNoise:   blueNoiseView,
			Sky:         scene.sky.WithSampler(sampler),
			Proj:        proj,
			ProjInverse: proj.Inverse(),
			View:        view,
			ViewInverse: viewInverse,

			ViewProj: proj.Mul4x4(view),
		}
	}

	// RT

	{
		out := dst.SubImage(
			dst.Dim(),
			vk.FORMAT_R8G8B8A8_UNORM,
			0, 1,
			0, 1).
			NewStorageView()
		defer jq.Cleanup(out.Destroy)

		pipeline, sbt := pathTracer()

		args := struct {
			Camera      gpu.Pointer[_FrameData]
			TLAS        gpu.UnsafePointer
			Out         gpu.StorageView
			TestTexture gpu.SamplingViewWithSampler
		}{
			Camera:      frameData,
			TLAS:        scene.tlas,
			Out:         out,
			TestTexture: testTexture.View.WithSampler(sampler),
		}
		gpu.EnqueueTraceRays(jq, pipeline, &sbt, res.X, res.Y, 1, &args)
	}

	// TODO: overlays like HUD, damage numbers, minimap, etc here

	// TODO: it would be interesting to update re on the device timeline
	re.ctr++
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
