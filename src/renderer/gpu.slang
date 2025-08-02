[require(spirv)] module;

[[vk::binding(0, 0)]] __DynamicResource<__DynamicResourceKind.Sampler> samplers[];
[[vk::binding(1, 0)]] __DynamicResource<__DynamicResourceKind.General> sampledImages[];
[[vk::binding(2, 0)]] __DynamicResource<__DynamicResourceKind.General> storageImages[];

public struct Slice<T> {
    public T *data; // TODO: make private
    uint64_t len;
    uint64_t cap;

	// TODO: can we somehow allow taking a pointer?
    public __subscript(int i) -> T
    {
		// TODO: do bounds checking
        get { return data[i]; }
        set { data[i] = newValue; }
    }
};

public struct Accel {
    public uint64_t ptr; // TODO: make private

    // public void TraceRay()
};

struct imageDescriptors {
	uint32_t sampling;
	uint32_t loadStore;
};

// TODO: *Views are deprecated in favor of this. We'll need to cook something
// up for samplers.
public struct Image {
	void *base;
	imageDescriptors descriptors;

	// Image()

	[require(spirv, nonuniformqualifier)]
	public float4 Load2D(int2 P) {
		return storageImages[NonUniformResourceIndex(descriptors.loadStore)].asOpaqueDescriptor<RWTexture2D>()[P];
	}

	[require(spirv, nonuniformqualifier)]
	public void Store2D(int2 P, float4 v) {
		storageImages[NonUniformResourceIndex(descriptors.loadStore)].asOpaqueDescriptor<RWTexture2D>()[P] = v;
	}
};

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
		sampledImages[NonUniformResourceIndex(handle)].asOpaqueDescriptor<Texture2D>().GetDimensions(extent.x, extent.y);
		return extent;
	}

	[require(spirv, nonuniformqualifier)]
	public float4 At2D(int2 P) {
		return sampledImages[NonUniformResourceIndex(handle)].asOpaqueDescriptor<Texture2D>().Load(int3(P, 0));
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
		return sampledImages[NonUniformResourceIndex(Image().handle)].asOpaqueDescriptor<Texture2D>().Sample(samplers[NonUniformResourceIndex(Sampler().handle >> 20)].asOpaqueDescriptor<SamplerState>(), P);
    }

    [require(spirv, nonuniformqualifier)]
    public float4 Sample2DLOD(float2 P, float level) { // rename to Sample2DF? Or should F be default so we do a Sample2DU for uint and Sample2DI for int?
        return sampledImages[NonUniformResourceIndex(Image().handle)].asOpaqueDescriptor<Texture2D>().Sample(samplers[NonUniformResourceIndex(Sampler().handle >> 20)].asOpaqueDescriptor<SamplerState>(), P);
    }

	[require(spirv, nonuniformqualifier)]
	public float4 SampleCube(float3 P) {
		return sampledImages[NonUniformResourceIndex(Image().handle)].asOpaqueDescriptor<TextureCube>().Sample(samplers[NonUniformResourceIndex(Sampler().handle >> 20)].asOpaqueDescriptor<SamplerState>(), P);
	}

	[require(spirv, nonuniformqualifier)]
	public float4 SampleCubeLOD(float3 P, float level) {
		return sampledImages[NonUniformResourceIndex(Image().handle)].asOpaqueDescriptor<TextureCube>().SampleLevel(samplers[NonUniformResourceIndex(Sampler().handle >> 20)].asOpaqueDescriptor<SamplerState>(), P, level);
	}
};

public struct StorageView {
    internal uint32_t handle;

    __init(uint32_t x) { handle = x; }

	[require(spirv, nonuniformqualifier)]
	public float4 Load2D(int2 P) {
		return storageImages[NonUniformResourceIndex(handle)].asOpaqueDescriptor<RWTexture2D>()[P];
	}

	[require(spirv, nonuniformqualifier)]
	public void Store2D(int2 P, float4 v) {
		storageImages[NonUniformResourceIndex(handle)].asOpaqueDescriptor<RWTexture2D>()[P] = v;
	}
};
