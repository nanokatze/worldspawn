package postprocess

// TODO: move different postprocessing operators into subpackages?

import (
	_ "embed"
	"structs"
	"sync"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

//go:generate slangc -target spirv -profile spirv_1_6 -fvk-use-entrypoint-name -fvk-use-c-layout -matrix-layout-row-major -capability vk_mem_model -capability spvDescriptorHeapEXT -o _shaders.spv bloom.slang

//go:embed _shaders.spv
var _shaders []byte

// TODO: allow the user to provide temporary storage
func Bloom(jq *gpu.JobQueue, dst, src *gpu.Image) {
	extent := [2]int(src.Extent())
	tmp := gpu.NewImage(
		gpu.MakeImageConfig(vk.FORMAT_E5B9G9R9_UFLOAT_PACK32, extent[:]).WithMips(7), // TODO: compute it from something?
		gpu.ImageWithUsage(vk.IMAGE_USAGE_SAMPLED_BIT|vk.IMAGE_USAGE_STORAGE_BIT))
	defer jq.Cleanup(tmp.Destroy)

	tmpMips := make([]*gpu.Image, tmp.Mips())
	for i := range tmpMips {
		tmpMips[i] = tmp.SubImage(gpu.SliceMips{i, i + 1})
		defer jq.Cleanup(tmpMips[i].Destroy)
	}

	tmp.EnqueueInit(jq)

	// downsample
	lastMip := src
	for i := 1; i < tmp.Mips(); i++ {
		tmpMip := tmpMips[i]
		bloomDownsample(jq, tmpMip, lastMip, [2]int(tmpMip.Extent()))
		lastMip = tmpMip
	}

	// upsample
	for i := 1; i < tmp.Mips(); i++ {
		radius := float32(1)
		if i > 1 {
			radius = 0.5
		}
		bloomUpsample(jq, dst, tmpMips[i], extent, radius)
	}
}

// TODO: all passes should delegate memory to the user so that the user can take
// care of aliasing to reduce storage costs.

type bloomDownsampleShaderEnv struct {
	_                 structs.HostLayout
	OutImage          gpu.ImageDescriptor
	InImage           gpu.ImageDescriptor
	Extent            [2]uint32
	SamplerDescriptor gpu.ImageSampler
}

var bloomDownsampleShader = lazySpvFileComputeShader[bloomDownsampleShaderEnv]{entry: "bloomDownsample"}

var downsampleSampler = sync.OnceValue(func() gpu.ImageSampler {
	return gpu.NewSampler(&vk.SamplerCreateInfo{
		SType:        vk.STRUCTURE_TYPE_SAMPLER_CREATE_INFO,
		AddressModeU: vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
		AddressModeV: vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
		MagFilter:    vk.FILTER_LINEAR,
		MinFilter:    vk.FILTER_LINEAR,
		MipmapMode:   vk.SAMPLER_MIPMAP_MODE_LINEAR,
		MaxLod:       9999,
	})
})

// TODO: temporary storage should also be up to the user.
// TODO: make this private
func bloomDownsample(jq *gpu.JobQueue, out, in *gpu.Image, extent [2]int) {
	dim := extent
	for i := range 2 {
		dim[i] = (dim[i] + 16 - 1) / 16
	}

	gpu.ParallelFor(jq, dim[:],
		bloomDownsampleShader.Bind(bloomDownsampleShaderEnv{
			OutImage: out.Descriptor(),
			InImage:  in.Descriptor(),
			Extent: [2]uint32{
				uint32(extent[0]),
				uint32(extent[1]),
			},
			SamplerDescriptor: downsampleSampler(),
		}))
}

type bloomUpsampleShaderEnv struct {
	_                 structs.HostLayout
	AccImage          gpu.ImageDescriptor
	InImage           gpu.ImageDescriptor
	Extent            [2]uint32
	SamplerDescriptor gpu.ImageSampler
	Radius            float32
}

var bloomUpsampleShader = lazySpvFileComputeShader[bloomUpsampleShaderEnv]{entry: "bloomUpsample"}

var upsampleSampler = sync.OnceValue(func() gpu.ImageSampler {
	return gpu.NewSampler(&vk.SamplerCreateInfo{
		SType:        vk.STRUCTURE_TYPE_SAMPLER_CREATE_INFO,
		AddressModeU: vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
		AddressModeV: vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
		MagFilter:    vk.FILTER_LINEAR,
		MinFilter:    vk.FILTER_LINEAR,
		MipmapMode:   vk.SAMPLER_MIPMAP_MODE_LINEAR,
		MaxLod:       9999,
	})
})

func bloomUpsample(jq *gpu.JobQueue, acc, in *gpu.Image, extent [2]int, radius float32) {
	dim := extent
	for i := range 2 {
		dim[i] = (dim[i] + 16 - 1) / 16
	}

	gpu.ParallelFor(jq, dim[:],
		bloomUpsampleShader.Bind(bloomUpsampleShaderEnv{
			AccImage: acc.Descriptor(),
			InImage:  in.Descriptor(),
			Extent: [2]uint32{
				uint32(extent[0]),
				uint32(extent[1]),
			},
			SamplerDescriptor: upsampleSampler(),
			Radius:            radius,
		}))
}

type lazySpvFileComputeShader[T any] struct {
	once sync.Once

	shader *gpu.ComputeShader[T]

	entry string
}

func (lazyShader *lazySpvFileComputeShader[T]) Bind(env T) gpu.ComputeClosure[T] {
	lazyShader.once.Do(func() {
		lazyShader.shader = gpu.CompileComputeShader[T](_shaders, lazyShader.entry)
	})
	return lazyShader.shader.Bind(env)
}
