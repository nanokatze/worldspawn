import gpu;
import transfer_functions;
import scene;
import material_interpreter;
import internal.lds;
import sampling;
import spectrum;
import bxdf;
import bsdf_diffuse;
import attribute;

struct ColorPayload {
    // Light transmitted along the path
    Spectrum L;

    // Path throughput
    Spectrum beta;

    float3 raydir;

    // TODO: we also need to shove p_b here

    // Whether this path has bounced off non-vampire geometry.
    bool antiVampirePath;

    // bool specularBounce;
};

// TODO: move this to sampling.slang or something idk
static const let Bounces = 2;

enum Dimension {
    X,
    Y,
    Z,
    W,
    Num,
};

struct FrameData {
    uint32_t frameNumber; // TODO: rename

    SamplingView blueNoise;

    // int bounces;

    float maxRayDist;

    matrix<float, 4, 4> proj;
    matrix<float, 4, 4> projInverse;
    matrix<float, 4, 4> view;
    matrix<float, 4, 4> viewInverse;

    // Precomputed stuff goes at the end

    matrix<float, 4, 4> viewProj; // precomputed extras should always go at the end
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

float getLDSValue(int bounce, int dimension) {
    let lds = Roberts(Dimension::Num * Bounces);
    let point = push.camera->frameNumber;

    return lds.Get(bounce * Bounces + dimension, point);
}

[[shader("raygeneration")]]
void raygen() {
    uint3 LaunchID = DispatchRaysIndex();
    uint3 LaunchSize = DispatchRaysDimensions();

    float2 pixelCenter = float2(LaunchID.xy) + float2(0.5, 0.5);
    // TODO: the host should fill this in.
    // float2 pixelCenter = float2(LaunchID.xy) + float2(getLDSValue(0, 0), getLDSValue(0, 1));
    float2 inUV = pixelCenter / float2(LaunchSize.xy);
    float2 d = inUV * 2.0 - 1.0;
    float4 target = mul(push.camera->projInverse, float4(d.x, d.y, 1, 1));

    // TODO: special case the first bounce, we need to get the primary surface
    // for denoiser and motion vectors/mask out for TAA

    ColorPayload payload = {};
    payload.L = Spectrum(0.0);
    payload.beta = Spectrum(1.0);
    payload.antiVampirePath = false;
    // payload.specularBounce = false;

    RayDesc rayDesc;
    rayDesc.Origin = mul(push.camera->viewInverse, float4(0, 0, 0, 1)).xyz;
    // TODO: do we actually need normalize here?
    rayDesc.Direction = normalize(mul(push.camera->viewInverse, float4(normalize(target.xyz), 0)).xyz);
    rayDesc.TMin = 0.0;
    rayDesc.TMax = push.camera->maxRayDist; // TODO: make these user-configurable

    // TODO: make the number of bounces configurable by host

    for (int bounce = 0; bounce < 2; bounce++) {
        RayQuery<RAY_FLAG_FORCE_OPAQUE> rq;
        rq.TraceRayInline(
            rtas(push.scene->accel.ptr),
            0, // OR'd with flags above
            payload.antiVampirePath ? 0b10 : 0b01,
            rayDesc);
        rq.Proceed();

        if (rq.CommittedStatus() == COMMITTED_TRIANGLE_HIT) {
            let hitpos = rq.WorldRayOrigin() + rq.WorldRayDirection() * max(rq.CommittedRayT() - 0.001, 0);

            closesthit(
                bounce,
                LaunchID,
                rq.CommittedInstanceIndex(),
                rq.CommittedGeometryIndex(),
                rq.CommittedPrimitiveIndex(),
                hitpos,
                rq.WorldRayDirection(),
                rq.CommittedObjectToWorld3x4(),
                payload,
                rq.CommittedTriangleBarycentrics());

            if (bounce > 0) {
                // TODO: introduce Spectrum.MaxComponentValue()
                float q = max(0, 1.0 - max(max(payload.beta.x, payload.beta.y), payload.beta.z));
                // TODO: use an ordinary RNG here
                if (q <= getLDSValue(bounce, 0))
                    break;
                payload.beta /= 1.0 - q;
            }

            rayDesc.Origin = hitpos;
            rayDesc.Direction = payload.raydir;
        } else if (rq.CommittedStatus() == COMMITTED_NOTHING) {
            // TODO: this is miss code, it should live in its own func

            float3 V = rq.WorldRayDirection();

            // TODO: make a matrix to convert from our convention to KTX2 convention
            // TODO: use SampleCubeGrad. We can use aniso sampling for cube textures.
            // Actually, aniso sampling is broken across many impls. We might wanna look
            // into filtering after shading for sky as well, but hopefully things might
            // get fixed.
            payload.L += payload.beta * push.scene->lightSampler.env.SampleCubeLOD(float3(-V.y, V.z, V.x), 0).xyz;

            break;
        } else {
            /* unreachable */
            break;
        }
    }

    // TODO: pack if we're storing into e5rgb9
    push.out.Store2D((int2)LaunchID.xy, float4(oetf(0, payload.L), 1));
}

// TODO: change float3 barys to be barys_yz
float
interpolate1(float3x1 attrs, float3 barys)
{
    return mul(transpose(attrs), barys);
}

float2
interpolate2(float3x2 attrs, float2 barys)
{
    return mul(transpose(attrs), float3(1.0 - barys.x - barys.y, barys));
}

float3
interpolate3(float3x3 attrs, float2 barys)
{
    return mul(transpose(attrs), float3(1.0 - barys.x - barys.y, barys));
}

struct Fetcher {
    uint32_t *pc;

    [[mutating]]
    uint32_t fetch() {
        return *pc++;
    }
};

struct Registers {
    // TODO: place the first few regs in actual hw regs and the rest into
    // scratch.
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
};

struct ShaderRecordData {
    uint32_t *code;
};

[[vk::shader_record]] ConstantBuffer<ShaderRecordData> srd;

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

struct composedBxDF /* : BxDF */ {
    Registers regs; // it would be nice if this could be just a pointer...
    // TODO: a table that describes how to interpreter regs
};

void closesthit(
    int bounce,
    uint3 dispatchRaysIndex,
    uint instanceIndex,
    uint geometryIndex,
    uint primitiveIndex,
    float3 hitpos,
    float3 WorldRayDirection,
    float3x4 ObjectToWorld,
    inout ColorPayload payload, in float2 barys_yz)
{
    // For an optimization we could check payload.antiVampirePath. If it's set,
    // there's no way this could be a vampire instance, so we don't need to
    // check the visibility mask.
    bool nonVampire = ((push->scene->accelInstances[instanceIndex].instanceIDAndMask >> 24) & 0b10) != 0;

    let bntex = push.camera->blueNoise;
    let bn = push.camera->blueNoise.At2D(dispatchRaysIndex.xy & (bntex.Extent2D() - 1)).xyz;

    let materialParams = &push.scene->materialParams.data[instanceIndex * push.scene->maxPartsPerMesh + geometryIndex];

    // TODO: pass the entire thing from host, rather than constructing it here.
    IndexSlice idxs = {(uint8_t*)materialParams->triangles, (uint64_t)2 << 62};

    let triangleIndices = idxs.Load3(primitiveIndex);

    Fetcher fetcher;
    fetcher.pc = materialParams->code;

    Registers regs;

    // TODO: factor decoding out
evalLoop:
    for (;;) {
        let instrw = fetcher.fetch();

        let op = (Op)(instrw & 0xff);
        let dst = (instrw >> 8) & 0xff;
        let src0 = (instrw >> 16) & 0xff;
        let src1 = (instrw >> 24) & 0xff;

        // TODO: reorganize instr encoding to be hierarchical, with the first
        // step being basically "instruction template" and ops following. So
        // e.g. we'd have a enum Opc and we'd have the following Opcs: Stop,
        // DstImm32, DstOneSrc, DstTwoSrc, PredTwoSrc, ...
        switch (op) {
        default:
            // TODO: default should abort()
        case Op::Stop:
            break evalLoop;

        case Op::Movk: {
            uint32_t val = fetcher.fetch();
            regs[dst] = val;
            break;
        }

        case Op::AddF32: {
            regs[dst] = reinterpret<uint32_t>(
                reinterpret<float>(regs[src0]) +
                reinterpret<float>(regs[src1]));
            break;
        }
        case Op::SubF32: {
            regs[dst] = reinterpret<uint32_t>(
                reinterpret<float>(regs[src0]) -
                reinterpret<float>(regs[src1]));
            break;
        }
        case Op::MinF32: {
            regs[dst] = reinterpret<uint32_t>(min(
                reinterpret<float>(regs[src0]),
                reinterpret<float>(regs[src1])));
            break;
        }

        case Op::FracF32: {
            regs[dst] = reinterpret<uint32_t>(frac(reinterpret<float>(regs[src0])));
            break;
        }

        case Op::LessOrEqualF32: {
            regs[dst] = reinterpret<float>(regs[src0]) <= reinterpret<float>(regs[src1]);
            break;
        }

        case Op::ConditionalSelect32: {
            uint32_t predicate = fetcher.fetch();
            regs[dst] = regs[predicate] ? regs[src0] : regs[src1];
            break;
        }

        case Op::Load: {
            uint32_t off = fetcher.fetch();
            regs[dst] = *(uint32_t*)((uint8_t*)materialParams + off);
            break;
        }
        case Op::LoadAttribute: {
            uint32_t off = fetcher.fetch();
            float2 *attrbuf = *(float2**)((uint8_t*)materialParams + off);
            float2 attr0 = attrbuf[triangleIndices[0]];
            float2 attr1 = attrbuf[triangleIndices[1]];
            float2 attr2 = attrbuf[triangleIndices[2]];
            // TODO: make a dedicated interpolation function which will take care of the
            // edge cases
            float2 attr = interpolate2(float3x2(attr0, attr1, attr2), barys_yz);
            regs[dst+0] = reinterpret<uint32_t>(attr.x);
            regs[dst+1] = reinterpret<uint32_t>(attr.y);
            // TODO: we'll probs be loading the entire vec3 or vec4 at all times
            // I guess?
            break;
        }
        }
    }

    // TODO: introduce a bxdf type which will represent composed bxdf in terms
    // of Registers and a table describing which bxdfs to compose and where in
    // the registers the inputs and weights lie. We'll use this to neatly
    // represent bxdfs composed "dynamically".

    let color = reinterpret<float3>(uint3(regs[0], regs[1], regs[2]));
    // TODO: get this from regs as well
    let normal = mul((float3x3)ObjectToWorld, // ns
        interpolate3(
            float3x3(
                materialParams->normals[triangleIndices[0]],
                materialParams->normals[triangleIndices[1]],
                materialParams->normals[triangleIndices[2]]),
            barys_yz));

    let shadingFrame = Frame::fromNormal(normal);

    let wo = shadingFrame.worldToLocal(-WorldRayDirection);

    Diffuse bsdf = {
        color,
        // 0.0,
    };

    let lightSampler = &push.scene->lightSampler;

    if ((bsdf.Flags() & (BSDFFlags::Diffuse | BSDFFlags::Glossy)) &&
        lightSampler->emissiveInstanceCount > 0) {
        float u = fmod(bn.z + getLDSValue(bounce, 2), 1.0);

        float2 b = fmod(bn.yx + float2(getLDSValue(bounce, 0), getLDSValue(bounce, 1)), 1.0);

        float asd = fmod(bn.z + getLDSValue(bounce, 3), 1.0);

        // TODO: encapsulate this into lightSampler->SampleLight
        let instanceIndex = (int)(asd * (float)lightSampler->emissiveInstanceCount);
        let instance = &lightSampler->emissiveInstances.data[instanceIndex];

        let instancePdf = 1.0 / (float)lightSampler->emissiveInstanceCount;

        if (instance->indexBuffer.data) {
        let triPdf = 1.0 / (float)instance->indexBuffer.Len();
        // TODO: we need to make sure this actually triangle < len always
        let triIdx = (int)((float)instance->indexBuffer.Len() * u);
        let vertIdxs = instance->indexBuffer[triIdx];

        light l = {
            {
                mul(instance->transform, float4(instance->posBuffer[vertIdxs[0]], 1.0)),
                mul(instance->transform, float4(instance->posBuffer[vertIdxs[1]], 1.0)),
                mul(instance->transform, float4(instance->posBuffer[vertIdxs[2]], 1.0)),
            },
            Spectrum(10.0),
        };

        let ls = l.Sample(hitpos, b);

        RayDesc rd;
        rd.Origin = hitpos;
        rd.Direction = ls.wi;
        rd.TMin = 0.0;
        // TODO: ls.dist can be inf for infinite lights, so we should clamp it
        // to max ray length
        // TODO: move this epsilon into a top level constant
        rd.TMax = max(ls.dist - 0.001, 0);
        RayQuery<RAY_FLAG_FORCE_OPAQUE> rq;
        rq.TraceRayInline(
            rtas(push.scene->accel.ptr),
            0, // OR'd with flags above
            nonVampire ? 0b10 : 0b01, // Bounces off non-vampire instances are always anti-vampire.
            rd);
        rq.Proceed();

        if (rq.CommittedStatus() == COMMITTED_NOTHING) {
            let pdfLight = ls.pdf * triPdf * instancePdf;

            let wi = shadingFrame.worldToLocal(ls.wi);

            let fLight = l.Li(b);
            let bsdfEvaluation = bsdf.Eval(wo, wi);

            let Ld = fLight * bsdfEvaluation.f * (abs(wi.z) /
                (pdfLight + bsdfEvaluation.pdf));

            payload.L += payload.beta * Ld;
        }
        }
    }

    let bsdfSample = bsdf.Sample(wo, getLDSValue(bounce, 0), fmod(bn.xy + float2(getLDSValue(bounce, 1), getLDSValue(bounce, 2)), 1.0));

    // TODO: once we get diffuse cache, (optionally) terminate the path for
    // diffuse-only bsdfSamples and instead get the information from the cache.

    payload.beta *= bsdfSample.f * abs(bsdfSample.wi.z) / bsdfSample.pdf;
    payload.raydir = shadingFrame.localToWorld(bsdfSample.wi);

    // Bounces that are reflected become anti-vampire if the surface was
    // non-vampire.
    //
    // Bounces that are transmitted stay vampire.
    if ((bsdfSample.flags & BSDFFlags::Reflected) && nonVampire) {
        payload.antiVampirePath = true;
    }
}
