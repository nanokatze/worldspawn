import gpu;

// TODO: properly rip this out of blender

struct bloomDownsampleShaderEnv {
    uint32_t outImage;
    uint32_t inImage;
    // uint2 dstOffset;
    // uint2 srcOffset;
    uint2 extent;
    uint32_t samplerDescriptor;
}

[[shader("compute"), numthreads(16, 16, 1)]]
void bloomDownsample(uniform bloomDownsampleShaderEnv env, uint3 threadIdx : SV_DispatchThreadID) {
    if (any(threadIdx.xy >= env.extent))
        return;

    const int2 Coords[13] = {
        int2( -1,  1 ), int2(  1,  1 ),
        int2( -1, -1 ), int2(  1, -1 ),

        int2(-2, 2), int2( 0, 2), int2( 2, 2),
        int2(-2, 0), int2( 0, 0), int2( 2, 0),
        int2(-2,-2), int2( 0,-2), int2( 2,-2)
    };

    const float Weights[13] = {
        // 4 samples
        // (1 / 4) * 0.5f = 0.125f
        0.125f, 0.125f,
        0.125f, 0.125f,

        // 9 samples
        // (1 / 9) * 0.5f
        0.0555555f, 0.0555555f, 0.0555555f,
        0.0555555f, 0.0555555f, 0.0555555f,
        0.0555555f, 0.0555555f, 0.0555555f
    };

    let dst = Image::fromLoadStoreDescriptor(env.outImage);
    let src = Image::fromSamplingDescriptor(env.inImage);

    let uv = (threadIdx.xy + 0.5) / env.extent;

    // TODO: downsample several levels in one dispatch with chungus workgroup
    // and a barrier()? We'd need to rewrite this to not use sampler. Also look
    // into using Halide.

    var sum = (float3)0;
    for (int i = 0; i < 13; i++) {
        sum += src.Sample2DLod(env.samplerDescriptor, uv + (float2)Coords[i] / src.Extent2D()).xyz * Weights[i];
    }
    dst.Store2D(threadIdx.xy, float4(sum, 1));
}

struct bloomUpsampleShaderEnv {
    uint32_t accumulatorImage;
    uint32_t inImage;
    uint2 extent;
    uint32_t samplerDescriptor;
    float radius;
}

[[shader("compute"), numthreads(16, 16, 1)]]
void bloomUpsample(uniform bloomUpsampleShaderEnv env, uint3 threadIdx : SV_DispatchThreadID) {
    if (any(threadIdx.xy >= env.extent))
        return;

    const float2 Coords[9] = {
        float2( -1.0f,  1.0f ), float2(  0.0f,  1.0f ), float2(  1.0f,  1.0f ),
        float2( -1.0f,  0.0f ), float2(  0.0f,  0.0f ), float2(  1.0f,  0.0f ),
        float2( -1.0f, -1.0f ), float2(  0.0f, -1.0f ), float2(  1.0f, -1.0f )
    };

    const float Weights[9] = {
        0.0625f, 0.125f, 0.0625f,
        0.125f,  0.25f,  0.125f,
        0.0625f, 0.125f, 0.0625f
    };

    let acc = Image::fromLoadStoreDescriptor(env.accumulatorImage);
    let src = Image::fromSamplingDescriptor(env.inImage);

    let uv = (threadIdx.xy + 0.5) / env.extent;

    var sum = (float3)0;
    for (int i = 0; i < 9; i++) {
        sum += src.Sample2DLod(env.samplerDescriptor, uv + (float2)Coords[i] / src.Extent2D()).xyz * Weights[i];
    }

    acc.Store2D(threadIdx.xy, float4(lerp(acc.Load2D(threadIdx.xy).xyz, sum, env.radius), 1));
}
