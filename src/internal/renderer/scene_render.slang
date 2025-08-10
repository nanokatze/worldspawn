import gpu;
import transfer_functions;

// TODO: for AttributeSlice will we want to split the header from the pointer?
// Or maybe not.

// Keep in sync with scene.go
static const let maxPartsPerMesh = 5;

enum RayType {
    RayTypeColor,
    RayTypeVisibility,
    NumRayTypes,
};

struct AttributeSlice {
    uint8_t *data;

    float4 Load(int i) {
        return float4(0);
    }
};

// TODO: delete this? We might want to be able to share index type for many
// index buffers. Especially so when it comes to clusters in CLASes.
struct IndexSlice {
    uint8_t *data;
    uint64_t aux; // length in bytes and index type, packed

    uint32_t Load(int i) {
        int index_type = (int)(aux >> 62);

        if (index_type == 0)
           return i;

        uint64_t stride = 1 << (index_type - 1);
        uint64_t off = i * stride;

        // TODO: have reasonable robustness behavior, or abort on out of bounds
        // and do the checking on startup

        uint8_t *p = data + off;
        switch (index_type) {
        case 1:
            return *(uint8_t*)p;
        case 2:
            return *(uint16_t*)p;
        case 3:
            return *(uint32_t*)p;
        default:
            // we should be able to put unreachable() here
            return i;
        }
    }

    uint3 Load3(int i) {
        return uint3(Load(3 * i + 0), Load(3 * i + 1), Load(3 * i + 2));
    }
};

struct MaterialParams {
    void *triangles;
    float3 *normals;
    float2 *uvs;
    SamplingViewWithSampler testTexture;
    float hmm[3];
};

struct Scene {
    SamplingViewWithSampler sky;

    Slice<MaterialParams> instances;
    Slice<uint8_t> accelInstances; // TODO: plop in proper AccelInstance
    Accel accel;

    Sampler sampler;
};

struct Camera {
    uint32_t frameNumber; // TODO: rename

    SamplingView blueNoise;

    matrix<float, 4, 4> proj;
    matrix<float, 4, 4> projInverse;
    matrix<float, 4, 4> view;
    matrix<float, 4, 4> viewInverse;

    // Precomputed stuff goes at the end

    matrix<float, 4, 4> viewProj; // precomputed extras should always go at the end
};

struct Push {
    Scene *scene;
    Camera *camera;
    StorageView out;
};

[[vk::push_constant]] Push push;

struct RayPayload {
    float3 color;
};

RaytracingAccelerationStructure rtas(uint64_t bda) {
    return spirv_asm { result:$$RaytracingAccelerationStructure = OpConvertUToAccelerationStructureKHR $bda };
}

[[shader("raygeneration")]]
void raygen() {
    uint3 LaunchID = DispatchRaysIndex();
    uint3 LaunchSize = DispatchRaysDimensions();

    const float2 pixelCenter = float2(LaunchID.xy) + float2(0.5, 0.5);
    const float2 inUV = pixelCenter / float2(LaunchSize.xy);
    float2 d = inUV * 2.0 - 1.0;
    float4 target = mul(push.camera->projInverse, float4(d.x, d.y, 1, 1));

    RayDesc rayDesc;
    rayDesc.Origin = mul(push.camera->viewInverse, float4(0, 0, 0, 1)).xyz;
    rayDesc.Direction = mul(push.camera->viewInverse, float4(normalize(target.xyz), 0)).xyz;
    rayDesc.TMin = 0.01;
    rayDesc.TMax = 69420.0;

    RayPayload payload = {};
    TraceRay(
        rtas(push.scene->accel.ptr),
        0,
        0b01,
        0, // sbtRecordOffset
        1, // sbtRecordStride
        0, // missIndex
        rayDesc,
        payload);

    // TODO: explore inlining the tail (everything after TraceRay) into hit/miss
    // shaders etc?

    // TODO: pack if we're storing into e5rgb9
    push.out.Store2D((int2)LaunchID.xy, float4(oetf(0, payload.color), 1));
}

// TODO: rename to skyMiss
[[shader("miss")]]
void sky(inout RayPayload rayPayload : SV_RayPayload) {
    float3 V = normalize(WorldRayDirection());

    // TODO: make a matrix to convert from our convention to KTX2 convention
    // TODO: use SampleCubeGrad. We can use aniso sampling for cube textures.
    // Actually, aniso sampling is broken across many impls. We might wanna look into filtering after shading for sky as well, but hopefully things might get fixed.
    rayPayload.color = push.scene->sky.SampleCubeLOD(float3(-V.y, V.z, V.x), 0).xyz;
}

float
interpolate1(float3x1 attrs, float3 barys)
{
    return mul(transpose(attrs), barys);
}

float2
interpolate2(float3x2 attrs, float3 barys)
{
    return mul(transpose(attrs), barys);
}

float3
interpolate3(float3x3 attrs, float3 barys)
{
    return mul(transpose(attrs), barys);
}

enum Op {
    Stop,

    Movk,

    AddF32,
    SubF32,
    MulF32,
    DivF32,
    MinF32,
    MaxF32,

    FracF32,

    EqualF32,
    NotEqualF32,
    LessOrEqualF32,

    ConditionalSelect,

    Load, // TODO: rename
    LoadAttribute, // TODO: rename

    BSDFOrenNayarDiffuse,
};

struct Fetcher {
    uint32_t *pc;

    [[mutating]]
    uint32_t fetch() {
        return *pc++;
    }
};

/*
struct Instr {
    Op op;
    uint32_t dst;
    uint32_t src0;
    uint32_t src1;
};

Instr decode(inout Fetcher fetcher) {
}
*/

struct Registers {
    // TODO: place the first few regs in actual hw regs and the rest into
    // scratch.
    // uint32_t regs[2];
    uint32_t scratch[32];

    __subscript(int i) -> uint32_t
    {
        get {
            return scratch[i];
        }

        set {
            scratch[i] = newValue;
        }
    }

    uint4 load4(int i) {
        return uint4(
            this[i+0],
            this[i+1],
            this[i+2],
            this[i+3],
        );
    }
};

struct ShaderRecordData {
    uint32_t *code;
};

[[vk::shader_record]] ConstantBuffer<ShaderRecordData> srd;

[[shader("closesthit")]]
void chit(inout RayPayload payload : SV_RayPayload, in float2 barys_yz : SV_IntersectionAttributes)
{
    let barys = float3(1.0 - barys_yz.x - barys_yz.y, barys_yz);

    let instance = &push.scene->instances.data[InstanceID() * maxPartsPerMesh + GeometryIndex()];

    // TODO: pass the entire thing from host, rather than constructing it here.
    IndexSlice idxs = {(uint8_t*)instance->triangles, (uint64_t)2 << 62};

    let triangleIndices = idxs.Load3(PrimitiveIndex());

    let V = normalize(WorldRayDirection());
    let L = normalize(float3(1, 1, 1));
    let N = mul((float3x3)ObjectToWorld(),
        interpolate3(float3x3(
            instance->normals[triangleIndices[0]],
            instance->normals[triangleIndices[1]],
            instance->normals[triangleIndices[2]]),
        barys));

    Fetcher fetcher;
    fetcher.pc = srd.code;

    Registers regs;

    // TODO: factor decoding out
outer:
    for (;;) {
        let op = (Op)fetcher.fetch();
        switch (op) {
        default:
            // TODO: default should abort()
        case Op::Stop:
            break outer;

        case Op::Movk: {
            uint32_t dst = fetcher.fetch();
            uint32_t val = fetcher.fetch();
            regs[dst] = val;
            break;
        }

        case Op::AddF32: {
            uint32_t dst = fetcher.fetch();
            uint32_t src0 = fetcher.fetch();
            uint32_t src1 = fetcher.fetch();
            regs[dst] = reinterpret<uint32_t>(
                reinterpret<float>(regs[src0]) +
                reinterpret<float>(regs[src1]));
            break;
        }
        case Op::SubF32: {
            uint32_t dst = fetcher.fetch();
            uint32_t src0 = fetcher.fetch();
            uint32_t src1 = fetcher.fetch();
            regs[dst] = reinterpret<uint32_t>(
                reinterpret<float>(regs[src0]) -
                reinterpret<float>(regs[src1]));
            break;
        }
        case Op::MinF32: {
            uint32_t dst = fetcher.fetch();
            uint32_t src0 = fetcher.fetch();
            uint32_t src1 = fetcher.fetch();
            regs[dst] = reinterpret<uint32_t>(min(
                reinterpret<float>(regs[src0]),
                reinterpret<float>(regs[src1])));
            break;
        }

        case Op::FracF32: {
            uint32_t dst = fetcher.fetch();
            uint32_t src = fetcher.fetch();
            regs[dst] = reinterpret<uint32_t>(frac(reinterpret<float>(regs[src])));
            break;
        }

        case Op::LessOrEqualF32: {
            uint32_t dst = fetcher.fetch();
            uint32_t src0 = fetcher.fetch();
            uint32_t src1 = fetcher.fetch();
            // TODO: write to predicate register instead
            regs[dst] = reinterpret<float>(regs[src0]) <= reinterpret<float>(regs[src1])
                ? reinterpret<uint32_t>((float)1.0)
                : reinterpret<uint32_t>((float)0.0);
            break;
        }

        case Op::ConditionalSelect: {
            uint32_t dst = fetcher.fetch();
            uint32_t cond = fetcher.fetch();
            uint32_t src0 = fetcher.fetch();
            uint32_t src1 = fetcher.fetch();
            // TODO: use the predicate register instead
            regs[dst] = regs[cond] ? regs[src0] : regs[src1];
            break;
        }

        case Op::Load: {
            uint32_t dst = fetcher.fetch();
            uint32_t off = fetcher.fetch();
            regs[dst] = *((uint32_t*)instance + off / 4);
            break;
        }
        case Op::LoadAttribute: {
            uint32_t dst = fetcher.fetch();
            uint32_t off = fetcher.fetch();
            float2 *attrbuf = *(float2**)((uint8_t*)instance + off);
            float2 attr0 = attrbuf[triangleIndices[0]];
            float2 attr1 = attrbuf[triangleIndices[1]];
            float2 attr2 = attrbuf[triangleIndices[2]];
            // TODO: make a dedicated interpolation function which will take care of the
            // edge cases
            float2 attr = interpolate2(float3x2(attr0, attr1, attr2), barys);
            regs[dst+0] = reinterpret<uint32_t>(attr.x);
            regs[dst+1] = reinterpret<uint32_t>(attr.y);
            // TODO: we'll probs be loading the entire vec3 or vec4 at all times
            // I guess?
            break;
        }

        case Op::BSDFOrenNayarDiffuse: {
            uint32_t src = fetcher.fetch();
            break;
        }
        }
    }

    float3 color = reinterpret<float3>(uint3(regs[0], regs[1], regs[2]));

    // plop this into BSDFOrenNayarDiffuse
    if (false) {
        // TODO: fire a shadow ray and stuff

        float roughness = 0.0;

        float LdotV = dot(L, V);
        float NdotL = dot(N, L);
        float NdotV = dot(N, V);

        float s = LdotV - NdotL * NdotV;
        float t = lerp(1.0, max(NdotL, NdotV), step(0.0, s));

        float sigma2 = roughness * roughness;
        float3 A = 1.0 + sigma2 * (color / (sigma2 + 0.13) + 0.5 / (sigma2 + 0.33));
        float B = 0.45 * sigma2 / (sigma2 + 0.09);

        float M_PI = 3.141592653589793;

        float3 result = color * max(0.0, NdotL) * (A + B * s / t) / M_PI;

        color = result;
    }

    payload.color = color;
}
