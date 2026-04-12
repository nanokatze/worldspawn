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
    public uint8_t *accel;
};

public struct ImageDescriptors {
	public uint32_t bits;

	[require(spirv, nonuniformqualifier)]
	public uint2 Extent2D() {
		uint32_t index = bits & ((1 << 20) - 1);
		uint2 extent;
		extent.x = 1000;
		extent.y = 1000;
		if ((bits & (1 << 20)) != 0) {
			sampledImages[NonUniformResourceIndex(index + 0)].asOpaqueDescriptor<Texture2D>().GetDimensions(extent.x, extent.y);
		} else if ((bits & (1 << 20)) != 0) {
			storageImages[NonUniformResourceIndex(index + 1)].asOpaqueDescriptor<RWTexture2D>().GetDimensions(extent.x, extent.y);
		} else {
			/* assert(0) */
		}
		return extent;
	}

	// TODO: all of these should take array index
	[require(spirv, nonuniformqualifier)]
	public float4 Load2D(int2 P) {
		uint32_t index = bits & ((1 << 20) - 1);
		return storageImages[NonUniformResourceIndex(index + 1)].asOpaqueDescriptor<RWTexture2D>()[P];
	}

	[require(spirv, nonuniformqualifier)]
	public void Store2D(int2 P, float4 v) {
		uint32_t index = bits & ((1 << 20) - 1);
		storageImages[NonUniformResourceIndex(index + 1)].asOpaqueDescriptor<RWTexture2D>()[P] = v;
	}

	[require(spirv, nonuniformqualifier)]
	public float4 Sample2DLod(uint32_t samplerDescriptor, float2 P) {
		uint32_t index = bits & ((1 << 20) - 1);
		return sampledImages[NonUniformResourceIndex(index + 0)].asOpaqueDescriptor<Texture2D>().SampleLevel(samplers[NonUniformResourceIndex(samplerDescriptor >> 20)].asOpaqueDescriptor<SamplerState>(), P, 0);
	}

	[require(spirv, nonuniformqualifier)]
	public float4 SampleCubeLOD(uint32_t samplerDescriptor, float3 P, float level) {
		uint32_t index = bits & ((1 << 20) - 1);
		return sampledImages[NonUniformResourceIndex(index + 0)].asOpaqueDescriptor<TextureCube>().SampleLevel(samplers[NonUniformResourceIndex(samplerDescriptor >> 20)].asOpaqueDescriptor<SamplerState>(), P, level);
	}

	[require(spirv, nonuniformqualifier)]
	public float4 Fetch2D(int2 P) {
		uint32_t index = bits & ((1 << 20) - 1);
		return sampledImages[NonUniformResourceIndex(index + 0)].asOpaqueDescriptor<Texture2DArray>().Load(int4(P, 0, 0));
	}
}
