[require(spirv)] module;

[[vk::binding(0, 0)]] __DynamicResource<__DynamicResourceKind.Sampler> samplers[];
[[vk::binding(1, 0)]] __DynamicResource<__DynamicResourceKind.General> sampledImages[];
[[vk::binding(2, 0)]] __DynamicResource<__DynamicResourceKind.General> storageImages[];

public struct Slice<T> {
    public T *data; // TODO: make private
    uint64_t len;
    uint64_t cap;

	// TODO: can we somehow allow taking a pointer?
	// TODO: this should take uint64_t index
    public __subscript(int i) -> T
    {
		// TODO: do bounds checking
        get { return data[i]; }
        set { data[i] = newValue; }
    }

	public uint64_t Len() { return len; }
};

public struct Accel {
    public uint64_t ptr; // TODO: make private
	public uint64_t size;

    // public void TraceRay()
};

public struct AccelInstance {
    public float transform[3][4]; // TODO: we could have this be a float3x4 or w/e
    public uint32_t instanceIDAndMask;
    public uint32_t sbtOffsetAndFlags;
    public uint8_t *accel; // TODO: replace with Accel when we make it match on host as well.
};

struct imageDescriptors {
	uint32_t sampling;
	uint32_t loadStore;
};

public struct ImageSampler {
	uint32_t descriptor;

	public static ImageSampler fromDescriptor(uint32_t descriptor) {
		ImageSampler s;
		s.descriptor = descriptor;
		return s;
	}
};

// TODO: *Views are deprecated in favor of this. We'll need to cook something
// up for samplers.
public struct Image {
	void *base;
	imageDescriptors descriptors;

	public static Image fromLoadStoreDescriptor(uint32_t LoadStoreDescriptor) {
		Image img;
		img.base = (void*)0;
		img.descriptors = {};
		img.descriptors.loadStore = LoadStoreDescriptor;
		return img;
	}

	public static Image fromSamplingDescriptor(uint32_t descriptor) {
		Image img;
		img.base = (void*)0;
		img.descriptors = {};
		img.descriptors.sampling = descriptor;
		return img;
	}

	[require(spirv, nonuniformqualifier)]
	public uint2 Extent2D() {
		uint2 extent;
		extent.x = 1000;
		extent.y = 1000;
		if (descriptors.sampling != 0) {
			sampledImages[NonUniformResourceIndex(descriptors.sampling)].asOpaqueDescriptor<Texture2D>().GetDimensions(extent.x, extent.y);
		} else if (descriptors.loadStore != 0) {
			storageImages[NonUniformResourceIndex(descriptors.loadStore)].asOpaqueDescriptor<RWTexture2D>().GetDimensions(extent.x, extent.y);
		} else {
			/* assert(0) */
		}
		return extent;
	}

	// TODO: all of these should take array index
	[require(spirv, nonuniformqualifier)]
	public float4 Load2D(int2 P) {
		return storageImages[NonUniformResourceIndex(descriptors.loadStore)].asOpaqueDescriptor<RWTexture2D>()[P];
	}

	[require(spirv, nonuniformqualifier)]
	public void Store2D(int2 P, float4 v) {
		storageImages[NonUniformResourceIndex(descriptors.loadStore)].asOpaqueDescriptor<RWTexture2D>()[P] = v;
	}

	[require(spirv, nonuniformqualifier)]
	public float4 Sample2DLod(uint32_t samplerDescriptor, float2 P) {
		return sampledImages[NonUniformResourceIndex(descriptors.sampling)].asOpaqueDescriptor<Texture2D>().SampleLevel(samplers[NonUniformResourceIndex(samplerDescriptor >> 20)].asOpaqueDescriptor<SamplerState>(), P, 0);
	}

	// Fetch2D or At2D?
	/*
	[require(spirv, nonuniformqualifier)]
	public float4 Fetch2D(int2 P) {
		return sampledImages[NonUniformResourceIndex(descriptors.sampling)].asOpaqueDescriptor<Texture2DArray>().Load(int4(P, 0, 0));
	}
	*/
};

/*
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

	// TODO: layer should come before P
	[require(spirv, nonuniformqualifier)]
	public float4 At2DArr(int2 P, int layer) {
		return sampledImages[NonUniformResourceIndex(handle)].asOpaqueDescriptor<Texture2DArray>().Load(int4(P, layer, 0));
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
*/