import gpu;
import transfer_functions;
import scene;
import material_interpreter;
import internal.lds;
import internal.interpreter;
import sampling;
import spectrum;
import bxdf;
import bsdf_diffuse;
import bsdf_microfacet;
import attribute;

// Shifted blue noise shamelessly stolen from rtx remix, which is
//
// Copyright (c) 2023-2024, NVIDIA CORPORATION. All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.  IN NO EVENT SHALL
// THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.
struct ShiftedBlueNoise<T_lds : LDS> {
    int2 initialShift;
    int index;

    T_lds lds; // TODO: move this elsewhere in the struct?

    __init(T_lds lds, int2 initialShift, int sampleOffset) {
        this.initialShift = initialShift;
        this.index = sampleOffset;
        this.lds = lds;
    }

    // TODO: blue noise is quantized so we only get like 256-1 values with
    // R8_UNORM (65536-1 with R16_UNORM) which might lead to bias. In practice
    // this already becomes a problem with light sampling, as we might have many
    // more emissive polys. We could augment the values with pure randomness.
    // For pure randomness we'd need to pull a per-frame random seed.

    [[mutating]]
    float Get1D() {
        return Get2D().x;
    }

    [[mutating]]
    float2 Get2D() {
        let index = this.index++;

        let point = float2(lds.Get(index, 0), lds.Get(index, 1));

        // TODO: make this a member on the ShiftedBlueNoise or an interface or
        // idk.
        let bn = push.camera->blueNoise;

        let value = push.camera->blueNoise.At2D((initialShift + int2(point * float2(bn.Extent2D()))) & (bn.Extent2D() - 1)).xy;

        return fract(value); // make sure we never return 1
    }
}

struct Path {
    // Light transmitted along the path
    Spectrum L;

    // Path throughput
    Spectrum beta;

    int bounce;

    // TODO: we also need to shove p_b here

    // Whether this path ...
    bool vampire;

    // bool specularBounce

    // Vertices of the next bounce
    float3 rayOrigin;
    float3 rayDir;
};

struct GBuffer {
    Path path;
    Spectrum albedo;
    float3 normal;
}

struct FrameData {
    uint32_t frameNumber; // TODO: rename

    SamplingView blueNoise;

    // int bounces;

    matrix<float, 4, 4> proj;
    matrix<float, 4, 4> projInverse;
    matrix<float, 4, 4> view;
    matrix<float, 4, 4> viewInverse;

    // Precomputed stuff goes at the end

    matrix<float, 4, 4> viewProj; // precomputed extras should always go at the end
}

// TODO: make things be methods on Push and rename Push to something else. Would
// be nice to inline FrameData in here.
struct Push {
    Scene *scene;
    FrameData *camera;
    StorageView out;
}

[[vk::push_constant]] Push push;

RaytracingAccelerationStructure rtas(uint64_t bda) {
    return spirv_asm { result:$$RaytracingAccelerationStructure = OpConvertUToAccelerationStructureKHR $bda };
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

    Path path = {};
    path.L = Spectrum(0.0);
    path.beta = Spectrum(1.0);
    path.bounce = 0;
    path.vampire = true;
    // path.specularBounce = false;
    path.rayOrigin = mul(push.camera->viewInverse, float4(0, 0, 0, 1)).xyz;
    // TODO: do we actually need normalize here?
    path.rayDir = normalize(mul(push.camera->viewInverse, float4(normalize(target.xyz), 0)).xyz);

    // TODO: make the number of bounces configurable by host

    // TODO: we should stop mid-way (after applying lights and stuff) so
    // path.bounce check should be moved into closesthit().
    for (path.bounce = 0; path.bounce < 2; path.bounce++) {
        RayDesc rayDesc;
        rayDesc.Origin = path.rayOrigin;
        rayDesc.Direction = path.rayDir;
        rayDesc.TMin = 0.0;
        rayDesc.TMax = 1.0 / 0.0;

        RayQuery<RAY_FLAG_FORCE_OPAQUE> rq;
        rq.TraceRayInline(
            rtas(push.scene->accel.ptr),
            0, // OR'd with flags above
            path.vampire ? 0b01 : 0b10,
            rayDesc);
        rq.Proceed();

        if (rq.CommittedStatus() == COMMITTED_TRIANGLE_HIT) {
            let hitpos = rq.WorldRayOrigin() + rq.WorldRayDirection() * max(rq.CommittedRayT() - 0.001, 0);

            closesthit(
                path,
                LaunchID,
                rq.CommittedInstanceIndex(),
                rq.CommittedGeometryIndex(),
                rq.CommittedPrimitiveIndex(),
                hitpos,
                rq.WorldRayDirection(),
                rq.CommittedObjectToWorld3x4(),
                rq.CommittedTriangleBarycentrics());

            if (path.bounce > 0) {
                // TODO: introduce Spectrum.MaxComponentValue()
                float q = max(0, 1.0 - max(max(path.beta.x, path.beta.y), path.beta.z));
                // TODO: use an ordinary RNG here
                if (0.1 < q)
                    break;
                path.beta /= 1.0 - q;
            }
        } else if (rq.CommittedStatus() == COMMITTED_NOTHING) {
            // TODO: this is miss code, it should live in its own func

            float3 V = rq.WorldRayDirection();

            // TODO: make a matrix to convert from our convention to KTX2 convention
            // TODO: use SampleCubeGrad. We can use aniso sampling for cube textures.
            // Actually, aniso sampling is broken across many impls. We might wanna look
            // into filtering after shading for sky as well, but hopefully things might
            // get fixed.
            // TODO: same as with emissive surfaces/volumes, we need to consult
            // lightSampler and apply the pdf (when lightSampler has a
            // possibility of returning the sky light.)
            path.L += path.beta * push.scene->lightSampler.env.SampleCubeLOD(float3(-V.y, V.z, V.x), 0).xyz;

            break;
        } else {
            /* unreachable */
            break;
        }
    }

    // TODO: pack if we're storing into e5rgb9
    push.out.Store2D((int2)LaunchID.xy, float4(oetf(0, path.L), 1));
}

// TODO: verts should be local. We also need to remember the original triIdxs
struct triangleLight {
    float3x4 ObjectToWorld;

    float3 verts[3];

    MaterialParams *materialParams;

    // TODO: switch to spherical triangle sampling for better distribution
    // instead.

    // Light emitted from a sampled point. u must be the same as what was used
    // for sampling the point.
    //
    // TODO: this should return Le, PDF and normal.
    leSample SampleLe(float3 hitpos, float2 u) {
        let area = length(cross(verts[1] - verts[0], verts[2] - verts[0])) * 0.5;
        let normal = normalize(cross(verts[1] - verts[0], verts[2] - verts[0]));

        let barys = SampleTriangle(u);

        let p = interpolate3(reinterpret<float3x3>(verts), barys);

        let uhh = p - hitpos;

        let wi = normalize(uhh);

        Registers regs;

        mateval(
            regs,
            materialParams->emissionCode,
            materialParams,
            ObjectToWorld,
            uint3(0),
            barys);

        return {
            spectrumFromRegs(regs, materialParams->outputsReg + 6),
            normal,
            dot(uhh, uhh) / (abs(dot(normal, wi)) * area),
        };
    }

    // Sample a point on the light.
    //
    // TODO: rename hitpos to something else
    // TODO: this should only choose a point on the light. Le, PDF and normal
    // should be factored out into the other function.
    lightSampleA SampleA(float3 hitpos, float2 u) {
        let barys = SampleTriangle(u);

        let p = interpolate3(reinterpret<float3x3>(verts), barys);

        let uhh = p - hitpos;

        let wi = normalize(uhh);

        return {
            wi,
            length(uhh),
        };
    }
};

float
interpolate1(float3x1 attrs, float2 barys)
{
    return mul(transpose(attrs), float3(1.0 - barys.x - barys.y, barys));
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

struct ShaderRecordData {
    uint32_t *code;
};

[[vk::shader_record]] ConstantBuffer<ShaderRecordData> srd;

/*
struct RegisterReader {
    Registers regs;
    int i;

    [[mutating]]
    uint32_t read() {
        return regs[i++];
    }
};
*/

// TODO: these should be using RegisterReader

Spectrum spectrumFromRegs(Registers regs, int i) {
    return Spectrum(reinterpret<float>(regs[i+0]),
                    reinterpret<float>(regs[i+1]),
                    reinterpret<float>(regs[i+2]));
}

Frame loadShadingFrameN(Registers regs, int i) {
    let normal = float3(reinterpret<float>(regs[i+0]),
                        reinterpret<float>(regs[i+1]),
                        reinterpret<float>(regs[i+2]));
    return Frame::fromNormal(normal);
}

BxDFAdapter<Diffuse> diffuseFromRegisters(Registers regs, int i) {
    return {
        loadShadingFrameN(regs, i+0),
        spectrumFromRegs(regs, i+3),
    };
}

enum bxdfid {
    diffuseBsdf,
}

// TODO: have a table of where each bxdf registers start?

struct bxdfspec {
    uint8_t id;
    uint8_t regoff;
};

struct composedbxdf {
    Registers regs;

    uint8_t contribregs;
};

/*
struct composedBxDF {
    // TODO: would be nice if we could put it behind a pointer tbh
    Registers regs;

    // should point to array of vec3s
    uint8_t contribregs;
    bxdfspec bxdfspecs[5]; // TODO: should be a global ptr
    uint8_t n;

    BSDFFlags Flags() {
        uint32_t result = 0;
        for (uint8_t i = 0; i < n; i++) {
            int regoff = bxdfspecs[i].regoff;

            switch ((bxdfid)bxdfspecs[i].id) {
            case bxdfid::diffuseBsdf:
                result |= (uint32_t)diffuseFromRegisters(regs, regoff).Flags();
                break;

            default:
                // abort();
                break;
            }
        }
        return (BSDFFlags)result;
    }

    BSDFEvaluatedResult Eval(float3 wo, float3 wi) {
        BSDFEvaluatedResult result = {};
        for (uint8_t i = 0; i < n; i++) {
            int regoff = bxdfspecs[i].regoff;
            float contrib = reinterpret<float>(regs[contribregs+i]);

            BSDFEvaluatedResult tmp = {};
            switch ((bxdfid)bxdfspecs[i].id) {
            case bxdfid::diffuseBsdf:
                tmp = diffuseFromRegisters(regs, regoff).Eval(wo, wi);
                break;

            default:
                // abort();
                break;
            }

            result = result.Add(tmp.Scale(contrib));
        }
        return result;
    }

    BSDFSample Sample(float3 wo, float uc, float2 u) {
        // TODO: we need to go over contribs until we encounter one that would
        // be > uc.
        uint8_t i = min(uc * (float)n, n-1); // TODO: this is horrid, actually

        int regoff = bxdfspecs[i].regoff;

        switch ((bxdfid)bxdfspecs[i].id) {
        case bxdfid::diffuseBsdf:
            return diffuseFromRegisters(regs, regoff).Sample(wo, uc, u);

        default:
            // abort();
            return {};
        }
    }
}
*/

// TODO: do not pass bounce and bn, encapsulate them into a sampler object
// TODO: do not pass bsdf here, make the caller deal with it instead
Spectrum sampleLd<T : BxDF>(
    bool vampire,
    float3 hitpos,
    float3 normal,
    float3 wo,
    T bsdf,
    float asd,
    float2 uLight,
) {
    let lightSampler = &push.scene->lightSampler;

    // TODO: encapsulate this into lightSampler->SampleLight
    let emissiveInstanceIndex = (int)(asd * (float)lightSampler->emissiveInstanceCount);
    let emissiveInstance = &lightSampler->emissiveInstances.data[emissiveInstanceIndex];

    float u = fract(asd * (float)lightSampler->emissiveInstanceCount);

    let instancePmf = 1.0 / (float)lightSampler->emissiveInstanceCount;

    if (emissiveInstance->originalInstanceIndex != 0) {
        let lightMaterialParams = &push.scene->materialParams.data[
            emissiveInstance->originalInstanceIndex * push.scene->maxPartsPerMesh +
            emissiveInstance->originalGeometryIndex];

        IndexSlice idxs2 = {(uint8_t*)lightMaterialParams->triangles, (uint64_t)2 << 62};

        let triPmf = 1.0 / (float)lightMaterialParams->num_triangles;
        // TODO: we need to make sure this actually triangle < len always
        let triIdx = (int)((float)lightMaterialParams->num_triangles * u);
        let vertIdxs = idxs2.Load3(triIdx);

        triangleLight l = {
            emissiveInstance->transform,
            {
                mul(emissiveInstance->transform, float4(lightMaterialParams->posBuffer[vertIdxs[0]], 1.0)),
                mul(emissiveInstance->transform, float4(lightMaterialParams->posBuffer[vertIdxs[1]], 1.0)),
                mul(emissiveInstance->transform, float4(lightMaterialParams->posBuffer[vertIdxs[2]], 1.0)),
            },
            lightMaterialParams,
        };

        let ls = l.SampleA(hitpos, uLight);

        RayDesc rd;
        rd.Origin = hitpos;
        rd.Direction = ls.wi;
        rd.TMin = 0.0;
        // TODO: move this epsilon into a top level constant
        rd.TMax = max(ls.dist - 0.001, 0);
        RayQuery<RAY_FLAG_FORCE_OPAQUE> rq;
        rq.TraceRayInline(
            rtas(push.scene->accel.ptr),
            0, // OR'd with flags above
            vampire ? 0b01 : 0b10,
            rd);
        rq.Proceed();

        if (rq.CommittedStatus() == COMMITTED_NOTHING) {
            let ls2 = l.SampleLe(hitpos, uLight);

            let pdfLight = ls2.pdf * triPmf * instancePmf;

            let fLight = ls2.Le;
            let bsdfEvaluation = bsdf.Eval(wo, ls.wi);

            // TODO: normal should be something per-bsdf rather than normal
            // because we don't know which normal it is.
            let Ld = fLight * bsdfEvaluation.f * (abs(dot(ls.wi, normal)) /
                (pdfLight + bsdfEvaluation.pdf));

            return Ld;
        }
    }

    return Spectrum(0);
}

// TODO: encapsulate this garbage into a nice Interaction struct or something
// like that.
void closesthit(
    __ref Path path,
    uint3 dispatchRaysIndex,
    uint instanceIndex,
    uint geometryIndex,
    uint primitiveIndex,
    float3 hitpos,
    float3 WorldRayDirection,
    float3x4 ObjectToWorld,
    float2 barys_yz)
{
    // For an optimization we could check path.antiVampire. If it's set, we
    // couldn't have hit a vampire instance, so we don't need to check the
    // visibility mask.
    bool vampire = ((push->scene->accelInstances[instanceIndex].instanceIDAndMask >> 24) & 0b11) == 0b01;

    // TODO: rename to sampler or something
    ShiftedBlueNoise<R2LDS> bn = ShiftedBlueNoise<R2LDS>(R2LDS(), dispatchRaysIndex.xy, 10 * path.bounce);

    let materialParams = &push.scene->materialParams.data[instanceIndex * push.scene->maxPartsPerMesh + geometryIndex];

    // TODO: pass the entire thing from host, rather than constructing it here.
    IndexSlice idxBuffer = {(uint8_t*)materialParams->triangles, (uint64_t)2 << 62};

    let triIdxs = idxBuffer.Load3(primitiveIndex);

    Registers regs;

    mateval(
        regs,
        materialParams->code,
        materialParams,
        ObjectToWorld,
        triIdxs,
        barys_yz);

    let wo = -WorldRayDirection;

    DiffuseEDF edf = {
        // TODO: we need a flag which specifies whether and where we have
        // emission in regs, rather than rely on emission-only program.
        materialParams->emissionCode
            ? spectrumFromRegs(regs, materialParams->outputsReg + 6)
            : Spectrum(0),
    };
    BxDFAdapter<Diffuse> bsdf = diffuseFromRegisters(regs, materialParams->outputsReg + 0);

    // bad, do not do this
    let normal = bsdf.shadingFrame.localToWorld(float3(0, 0, 1));

    // Account for emission
    let Le = edf.Eval(wo);
    if (any(Le > float3(0))) {
        // TODO: this is wrong, we need to apply MIS weight, which we'll need to
        // consult lightSampler for, to figure out the pdf.
        path.L += path.beta * Le;
    }

    // TODO: radiance cache sampling would fit into somewhere in here

    // Direct light sampling
    let u_Ld = float3(bn.Get1D(), bn.Get2D());
    if (bsdf.Flags() & (BSDFFlags::Diffuse | BSDFFlags::Glossy)) {
        let Ld = sampleLd(vampire, hitpos, normal, wo, bsdf, u_Ld.x, u_Ld.yz);

        if (any(Ld > float3(0)))
            path.L += path.beta * Ld;
    }

    // Sample the bsdf for the next ray

    let bsdfSample = bsdf.Sample(wo, bn.Get1D(), bn.Get2D());

    // TODO: once we get diffuse cache, (optionally) terminate the path for
    // diffuse-only bsdfSamples and instead get the information from the cache.

    path.beta *= bsdfSample.f * dot(bsdfSample.wi, normal) / bsdfSample.pdf;

    // Vampire bounces that are reflected become anti-vampire if the surface was
    // not vampire.
    //
    // Vampire bounces that are transmitted stay vampire.
    if ((bsdfSample.flags & BSDFFlags::Reflected) && !vampire) {
        path.vampire = false;
    }

    path.rayOrigin = hitpos;
    path.rayDir = bsdfSample.wi; // TODO: should we normalize() here?
}
