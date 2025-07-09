[require(spirv)] module;

[allow("parameterBindingsOverlap")] [[vk::binding(0, 0)]] SamplerState samplers[];

[allow("parameterBindingsOverlap")] [[vk::binding(1, 0)]] Texture2D sampledImages2D[];
[allow("parameterBindingsOverlap")] [[vk::binding(1, 0)]] TextureCube sampledImagesCube[];

[allow("parameterBindingsOverlap")] [[vk::binding(2, 0)]] RWTexture2D storageImageViews[];

public struct Sampler {
	internal uint32_t handle;

    __init(uint32_t x) { handle = x; }
};

// TODO: I honestly dislike these names, could we perhaps rename these to Image
// or whatever?
public struct SamplingView {
	internal uint32_t handle;

    __init(uint32_t x) { handle = x; }

	public SamplingViewWithSampler WithSampler(Sampler sampler) {
		return ::SamplingViewWithSampler(handle | sampler.handle);
	}

	// TODO: always use 3d types in these?

	[require(spirv, nonuniformqualifier)]
	public uint2 Extent2D() { // rename to Bounds2D? Size2D?
		uint2 extent;
		sampledImages2D[NonUniformResourceIndex(handle)].GetDimensions(extent.x, extent.y);
		return extent;
	}

	[require(spirv, nonuniformqualifier)]
	public float4 At2D(int2 P) {
		return sampledImages2D[NonUniformResourceIndex(handle)].Load(int3(P, 0));
	}
};

public struct SamplingViewWithSampler {
    internal uint32_t handle;

    __init(uint32_t x) { handle = x; }

	SamplingView Image() {
		return ::SamplingView(handle & 0x000fffff);
	}

	Sampler Sampler() {
		return ::Sampler(handle & 0xfff00000);
	}

	// TODO: get rid of implicit lod samples in favor of only explicit lod

	// TODO: always use float3 for point here?

	[require(spirv, nonuniformqualifier)]
	public float4 Sample2D(float2 P) { // rename to Sample2DF? Or should F be default so we do a Sample2DU for uint and Sample2DI for int?
		return sampledImages2D[NonUniformResourceIndex(Image().handle)].Sample(samplers[NonUniformResourceIndex(Sampler().handle >> 20)], P);
    }

    [require(spirv, nonuniformqualifier)]
    public float4 Sample2DLOD(float2 P, float level) { // rename to Sample2DF? Or should F be default so we do a Sample2DU for uint and Sample2DI for int?
        return sampledImages2D[NonUniformResourceIndex(Image().handle)].Sample(samplers[NonUniformResourceIndex(Sampler().handle >> 20)], P);
    }

	[require(spirv, nonuniformqualifier)]
	public float4 SampleCube(float3 P) {
		return sampledImagesCube[NonUniformResourceIndex(Image().handle)].Sample(samplers[NonUniformResourceIndex(Sampler().handle >> 20)], P);
	}

	[require(spirv, nonuniformqualifier)]
	public float4 SampleCubeLOD(float3 P, float level) {
		return sampledImagesCube[NonUniformResourceIndex(Image().handle)].SampleLevel(samplers[NonUniformResourceIndex(Sampler().handle >> 20)], P, level);
	}
};

public struct StorageView {
    internal uint32_t handle;

    __init(uint32_t x) { handle = x; }

	[require(spirv, nonuniformqualifier)]
	public float4 Load2D(int2 P) {
		return storageImageViews[NonUniformResourceIndex(handle)][P];
	}

	[require(spirv, nonuniformqualifier)]
	public void Store2D(int2 P, float4 v) {
		storageImageViews[NonUniformResourceIndex(handle)][P] = v;
	}
};
