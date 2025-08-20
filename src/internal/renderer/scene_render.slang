import gpu;
import transfer_functions;
import scene;
import material_interpreter;
import internal.lds;
import sampling;
import spectrum;

static const float M_PI = 3.141592653589793;

struct ColorPayload {
    float3 raydir;
    Spectrum attenuation;
};

// TODO: move this to sampling.slang or something idk
static const let Bounces = 1;

enum Dimension {
    X,
    Y,
    Z,
    Num,
};

struct FrameData {
    uint32_t frameNumber; // TODO: rename

    SamplingView blueNoise;

    matrix<float, 4, 4> proj;
    matrix<float, 4, 4> projInverse;
    matrix<float, 4, 4> view;
    matrix<float, 4, 4> viewInverse;

    // Precomputed stuff goes at the end

    matrix<float, 4, 4> viewProj; // precomputed extras should always go at the end
};

// This should be purely a description with no data associated.
//
// We should support everything we can get from blender.
struct AttributeDesc {
    int unused;
};

// TODO: we will sneak in the index type out of band so we'll construct it in
// the shader.
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

struct Push {
    Scene *scene;
    FrameData *camera;
    StorageView out;
};

[[vk::push_constant]] Push push;

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
    Spectrum attenuation = Spectrum(1.0);
    Spectrum L = float3(0.0);
    for (int i = 0; i < 2; i++) {
        RayQuery<RAY_FLAG_FORCE_OPAQUE> rq;
        rq.TraceRayInline(
            rtas(push.scene->accel.ptr),
            0, // OR'd with flags above
            0b01,
            rayDesc);
        rq.Proceed();

        ColorPayload payload = {};

        if (rq.CommittedStatus() == COMMITTED_TRIANGLE_HIT) {
            /* TODO: we should do a CallShader */
            chit2(
                LaunchID,
                rq.CommittedInstanceID(),
                rq.CommittedGeometryIndex(),
                rq.CommittedPrimitiveIndex(),
                rq.WorldRayDirection(),
                rq.CommittedObjectToWorld3x4(),
                payload,
                rq.CommittedTriangleBarycentrics());

            attenuation *= payload.attenuation;

            rayDesc.Origin = rq.WorldRayOrigin() + rq.WorldRayDirection() * rq.CommittedRayT();
            rayDesc.Direction = payload.raydir;
        } else {
            float3 V = rq.WorldRayDirection();

            // TODO: make a matrix to convert from our convention to KTX2 convention
            // TODO: use SampleCubeGrad. We can use aniso sampling for cube textures.
            // Actually, aniso sampling is broken across many impls. We might wanna look
            // into filtering after shading for sky as well, but hopefully things might
            // get fixed.
            L += attenuation * push.scene->sky.SampleCubeLOD(float3(-V.y, V.z, V.x), 0).xyz;

            break;
        }
    }

    // TODO: pack if we're storing into e5rgb9
    push.out.Store2D((int2)LaunchID.xy, float4(oetf(0, L), 1));
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

enum BSDFSampleFlags {
    // Absorbed = 0,
    Emitted = 1 << 0,
    Reflected = 1 << 1,
    Transmitted = 1 << 2,
    Diffuse = 1 << 3,
    Glossy = 1 << 4,
    Specular = 1 << 5,
};

struct BSDFSample {
    Spectrum f;
    float3 wi;
    float pdf; // rename to just pd
    BSDFSampleFlags flags;
    // float eta;
};

struct BSDFEvaluatedResult {
    Spectrum f;
    float pdf; // rename to just pd
};

interface BxDF {
    // BSDFSample Sample(float3 wo);
    // BSDFEvaluatedResult Evaluate(float3 wo, float3 wi);
};

float absCosTheta(float3 w) { return abs(w.z); }

// TODO: replace this with just a float3x3/Mat3x3
struct Frame {
  float3 x, y, z;

  static Frame fromNormal(float3 n) {
    float3 a = abs(n.x) > 0.5 ? float3(0, 0, 1) : float3(1, 0, 0);

    float3 b = normalize(cross(n, a));
    float3 t = cross(n, b);

    return {t, b, n};
  }

  static Frame fromNT(float3 n, float3 t, float sign = 1.0) {
    if (abs(dot(n, t)) > 0.9) return fromNormal(n);

    float3 b = normalize(cross(n, t)) * sign;
    t = cross(b, n);

    return {t, b, n};
  }

  float3 worldToLocal(float3 w) {
    return float3(dot(w, x), dot(w, y), dot(w, z));
  }

  float3 localToWorld(float3 l) {
    return x * l.x + y * l.y + z * l.z;
  }
};

struct OrenNayarDiffuse : BxDF {
    Spectrum R;
    float roughness;

    BSDFSample Sample(float3 wo, float uc, float2 u) {
        float3 wi = SampleCosineHemisphere(u);

        return {
            R / M_PI,
            wi,
            CosineHemispherePDF(absCosTheta(wi)),
            BSDFSampleFlags::Reflected,
        };
    }

    BSDFEvaluatedResult Evaluate(float3 wo, float3 wi) {
        if (wo.z < 1.5e-3f || wi.z < 1.5e-3f)
            return {};

        return {R / M_PI, CosineHemispherePDF(absCosTheta(wi))};
    }
};

void chit2(
    uint3 dispatchRaysIndex,
    uint instanceID,
    uint geometryIndex,
    uint primitiveIndex,
    float3 WorldRayDirection,
    float3x4 ObjectToWorld,
    inout ColorPayload payload, in float2 barys_yz)
{
    let bntex = push.camera->blueNoise;
    float3 bn = push.camera->blueNoise.At2D(dispatchRaysIndex.xy & (bntex.Extent2D() - 1)).xyz;

    LDS lds = Roberts(Dimension::Num * Bounces);
    let ldsPointIndex = push.camera->frameNumber;

    let barys = float3(1.0 - barys_yz.x - barys_yz.y, barys_yz);

    let instance = &push.scene->instances.data[instanceID * push.scene->maxPartsPerMesh + geometryIndex];

    // TODO: pass the entire thing from host, rather than constructing it here.
    IndexSlice idxs = {(uint8_t*)instance->triangles, (uint64_t)2 << 62};

    let triangleIndices = idxs.Load3(primitiveIndex);

    let wo = -WorldRayDirection;

    // This is supposed to be provided by the material
    let N = mul((float3x3)ObjectToWorld, // ns
        interpolate3(float3x3(
            instance->normals[triangleIndices[0]],
            instance->normals[triangleIndices[1]],
            instance->normals[triangleIndices[2]]),
        barys));

    Fetcher fetcher;
    fetcher.pc = instance->code;

    Registers regs;

    // TODO: factor decoding out
evalLoop:
    for (;;) {
        let op = (Op)fetcher.fetch();
        switch (op) {
        default:
            // TODO: default should abort()
        case Op::Stop:
            break evalLoop;

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
            uint32_t dst = fetcher.fetch();
            uint32_t src = fetcher.fetch();

            break;
        }
        }
    }

    float3 color = reinterpret<float3>(uint3(regs[0], regs[1], regs[2]));

    let frame = Frame::fromNormal(N);

    let woLocal = frame.worldToLocal(wo);

    OrenNayarDiffuse bsdf = {
        color,
        0.0,
    };

    let sample = bsdf.Sample(woLocal, lds.Get(0, ldsPointIndex), fmod(bn.xy + float2(lds.Get(1, ldsPointIndex), lds.Get(2, ldsPointIndex)), 1.0));

    // payload.color = bsdf.Evaluate(woLocal, sample.wi).f;
    payload.raydir = frame.localToWorld(sample.wi);
    payload.attenuation = sample.f * abs(sample.wi.z) / sample.pdf;

    // payload.color = color;
}
