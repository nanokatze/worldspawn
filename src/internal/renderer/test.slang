module;

import gpu;

struct Push {
    float2 offset;
    float2 extent;
    SamplingViewWithSampler tex;
};

[[vk::push_constant]] Push push;

static float2 vbo[] = {
    float2(0, 0),
    float2(0, 1),
    float2(1, 0),

    float2(1, 1),
    float2(1, 0),
    float2(0, 1),
};

struct VSOut {
    float4 pos : SV_Position;
    [[vk::location(0)]] float2 uv;
};

[shader("vertex")]
VSOut vertMain(uint32_t rawVertexID: SV_VulkanVertexID) {
    return VSOut(
        float4(-(float2)1 + 2 * (push.offset + vbo[rawVertexID] * push.extent), 0, 1),
        vbo[rawVertexID]
    );
}

struct FSOut {
    float4 out0 : SV_Target0;
};

[shader("fragment")]
FSOut fragMain(VSOut vsIn) {
    return FSOut(
        push.tex.Sample2DLOD(vsIn.uv, 0).xxxx
    );
}