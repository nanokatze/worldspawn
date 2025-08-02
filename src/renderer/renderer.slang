import gpu;
import transfer_functions;

// TODO: for AttributeSlice will we want to split the header from the pointer?
// Or maybe not.

// TODO: delete this? We might want to be able to share index type for many
// index buffers. Especially so when it comes to clusters in CLASes.
struct IndexSlice {
    uint8_t *data;
    uint64_t aux; // length in bytes and index type, packed

    uint32_t Load(int i) {
        int index_type = (int)(aux >> 62);

        // if (index_type == 0)
        //    return i;

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

// TODO: material interpreter!

struct Mesh {
    void *triangles;
    float2 *uvs;
    SamplingViewWithSampler testTexture;
    float hmm[3];
};

struct Scene {
	SamplingViewWithSampler sky;

    Slice<Mesh*> instances;
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

[vk::push_constant] Push push;

struct RayPayload {
    float3 color;
};

RaytracingAccelerationStructure rtas(uint64_t bda) {
    return spirv_asm { result:$$RaytracingAccelerationStructure = OpConvertUToAccelerationStructureKHR $bda };
}

[shader("raygeneration")]
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
        0,
        0,
        0,
        rayDesc,
        payload);

    // TODO: pack if we're storing into e5rgb9
    push.out.Store2D((int2)LaunchID.xy, float4(oetf(0, payload.color), 1));
}

// TODO: rename to skyMiss
[shader("miss")]
void sky(inout RayPayload rayPayload : SV_RayPayload) {
    float3 rayDir = normalize(WorldRayDirection());

    // TODO: make a matrix to convert from our convention to KTX2 convention
    // TODO: use SampleCubeGrad. We can use aniso sampling for cube textures.
    // Actually, aniso sampling is broken across many impls. We might wanna look into filtering after shading for sky as well, but hopefully things might get fixed.
    rayPayload.color = push.scene->sky.SampleCubeLOD(float3(-rayDir.y, rayDir.z, rayDir.x), 0).xyz;
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

/*
struct Test {
    int aaa;
};

[vk::shader_record] Test test;
*/

enum Op {
    Stop,
    Movk,
    Load,
    LoadAttribute,
    Frac,
    Sub,
    Min,
    LessOrEqual,
    CSel,
};

static uint32_t code[] = {
    (uint32_t)Op::LoadAttribute, 0,
    (uint32_t)Op::Frac, 0, 0,
    (uint32_t)Op::Frac, 1, 1,
    (uint32_t)Op::Movk, 6, reinterpret<uint32_t>((float)1.0),
    (uint32_t)Op::Sub, 2, 6, 0,
    (uint32_t)Op::Sub, 3, 6, 1,
    (uint32_t)Op::Min, 0, 0, 2,
    (uint32_t)Op::Min, 1, 1, 3,
    (uint32_t)Op::Min, 3, 0, 1,
    (uint32_t)Op::Movk, 4, reinterpret<uint32_t>((float)0.01),
    (uint32_t)Op::LessOrEqual, 5, 3, 4,
    (uint32_t)Op::Load, 0, 20,
    (uint32_t)Op::Load, 1, 24,
    (uint32_t)Op::Load, 2, 28,
    (uint32_t)Op::CSel, 0, 5, 6, 0,
    (uint32_t)Op::CSel, 1, 5, 6, 1,
    (uint32_t)Op::CSel, 2, 5, 6, 2,
    (uint32_t)Op::Stop,
};

[shader("closesthit")]
void chit(inout RayPayload payload : SV_RayPayload, in float2 barys_yz : SV_IntersectionAttributes)
{
    float3 barys = float3(1.0 - barys_yz.x - barys_yz.y, barys_yz);

    Mesh *instance = push.scene->instances[InstanceID()];

    // TODO: pass the entire thing from host, rather than constructing it here.
    IndexSlice idxs = {(uint8_t*)instance->triangles, (uint64_t)2 << 62};

    uint3 triangleIndices = idxs.Load3(PrimitiveIndex());

    float2 uv0 = instance->uvs[triangleIndices[0]];
    float2 uv1 = instance->uvs[triangleIndices[1]];
    float2 uv2 = instance->uvs[triangleIndices[2]];
    // TODO: make a dedicated interpolation function which will take care of the
    // edge cases
    float2 uv = interpolate2(float3x2(uv0, uv1, uv2), barys);

    // TODO: place the first few regs in actual hw regs and the rest into
    // scratch.
    uint32_t regs[32];

    int pc = 0;
    bool stop = false;
    while (!stop) {
        switch (code[pc++]) {
        default:
            // TODO: default should abort()
        case (uint32_t)Op::Stop:
            stop = true;
            break;
        case (uint32_t)Op::Load: {
            uint32_t dst = code[pc++];
            uint32_t off = code[pc++];
            regs[dst] = *((uint32_t*)instance + off / 4);
            break;
        }
        case (uint32_t)Op::Movk: {
            uint32_t dst = code[pc++];
            uint32_t val = code[pc++];
            regs[dst] = val;
            break;
        }
        case (uint32_t)Op::LoadAttribute: {
            uint32_t dst = code[pc++];
            regs[dst+0] = reinterpret<uint32_t>(uv.x);
            regs[dst+1] = reinterpret<uint32_t>(uv.y);
            break;
        }
        case (uint32_t)Op::Frac: {
            uint32_t dst = code[pc++];
            uint32_t src = code[pc++];
            regs[dst] = reinterpret<uint32_t>(frac(reinterpret<float>(regs[src])));
            break;
        }
        case (uint32_t)Op::Sub: {
            uint32_t dst = code[pc++];
            uint32_t src0 = code[pc++];
            uint32_t src1 = code[pc++];
            regs[dst] = reinterpret<uint32_t>(
                reinterpret<float>(regs[src0]) -
                reinterpret<float>(regs[src1]));
            break;
        }
        case (uint32_t)Op::Min: {
            uint32_t dst = code[pc++];
            uint32_t src0 = code[pc++];
            uint32_t src1 = code[pc++];
            regs[dst] = reinterpret<uint32_t>(min(
                reinterpret<float>(regs[src0]),
                reinterpret<float>(regs[src1])));
            break;
        }
        case (uint32_t)Op::LessOrEqual: {
            uint32_t dst = code[pc++];
            uint32_t src0 = code[pc++];
            uint32_t src1 = code[pc++];
            regs[dst] = reinterpret<float>(regs[src0]) <= reinterpret<float>(regs[src1])
                ? reinterpret<uint32_t>((float)1.0)
                : reinterpret<uint32_t>((float)0.0);
            break;
        }
        case (uint32_t)Op::CSel: {
            uint32_t dst = code[pc++];
            uint32_t cond = code[pc++];
            uint32_t src0 = code[pc++];
            uint32_t src1 = code[pc++];
            // TODO: regs[cond] should probs be a float, need to check blender's
            // semantics for this...
            regs[dst] = regs[cond] ? regs[src0] : regs[src1];
            break;
        }
        }
    }

    /*
    float v3 = reinterpret<float>(regs[0]);
    float3 v4 = v3 <= 0.01
        ? (float3)1.0
        : float3(instance->hmm[0], instance->hmm[1], instance->hmm[2]);
    */

    payload.color = reinterpret<float3>(uint3(regs[0], regs[1], regs[2]));
}

[shader("closesthit")]
void chit2(inout RayPayload payload : SV_RayPayload, in float2 barys_yz : SV_IntersectionAttributes)
{
    payload.color = (float3) {0.0, 1.0, 0.0};
}
