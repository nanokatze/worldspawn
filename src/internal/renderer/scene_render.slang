import attribute;
import gpu;
import internal.integration.lds;
import internal.integration.mapping;
import internal.material.bsdf;
import internal.material.interpreter;
import scene;
import spectrum;
import transfer_functions;

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
//
// TODO: can we somehow move this into internal/integration?
struct ShiftedBlueNoise<T_lds : LDS> {
    T_lds lds; // TODO: move this elsewhere in the struct?
    int2 pixel;
    int sampleIndex;

    __init(T_lds lds, int2 pixel, int sampleOffset) {
        this.lds = lds;
        this.pixel = pixel;
        this.sampleIndex = sampleOffset;
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
        let sampleIndex = this.sampleIndex++;

        let point = float2(lds.Get(sampleIndex, 0), lds.Get(sampleIndex, 1));

        // TODO: make this a member on the ShiftedBlueNoise or an interface or
        // idk.
        let blueNoise = push.camera->blueNoise;
        let extent = blueNoise.Extent2D();

        let p = (pixel + int2(point * float2(extent))) & (extent - 1);

        let value = blueNoise.At2D(p).xy;

        return fract(value); // make sure we never return 1
    }
}

struct Path {
    // Light transmitted along the path
    Spectrum L;

    // Path throughput
    Spectrum beta;

    int depth;

    // TODO: we also need to shove p_b here

    // Whether this path ...
    // TODO: rename to something else
    bool vampire;

    // bool specularBounce

    // Vertices of the next bounce
    float3 rayOrigin;
    float3 rayDir;
};

struct FrameData {
    int maxBounces;

    int russianRouletteThreshold;

    SamplingView blueNoise;

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
    // TODO: buffers for normal, AOVs, etc
    StorageView out; // irradiance
}

[[vk::push_constant]] Push push;

RaytracingAccelerationStructure rtas(uint64_t bda) {
    return spirv_asm { result:$$RaytracingAccelerationStructure = OpConvertUToAccelerationStructureKHR $bda };
}

void exportIrradiance(Spectrum L) {
    uint3 LaunchID = DispatchRaysIndex();

    // TODO: pack if we're storing into e5rgb9
    push.out.Store2D((int2)LaunchID.xy, float4(oetf(0, L), 1));
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
    path.depth = 0;
    path.vampire = true;
    // path.specularBounce = false;
    path.rayOrigin = mul(push.camera->viewInverse, float4(0, 0, 0, 1)).xyz;
    // TODO: do we actually need normalize here?
    path.rayDir = normalize(mul(push.camera->viewInverse, float4(normalize(target.xyz), 0)).xyz);

    for (;;) {
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

            let stop = !closesthit(
                path,
                LaunchID,
                rq.CommittedInstanceIndex(),
                rq.CommittedGeometryIndex(),
                rq.CommittedPrimitiveIndex(),
                hitpos,
                rq.WorldRayDirection(),
                rq.CommittedObjectToWorld3x4(),
                rq.CommittedTriangleBarycentrics());
            if (stop)
                break;
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
            /* abort(); */
            break;
        }
    }

    exportIrradiance(path.L);
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

        EvaluatedInterpretedMaterial m;
        m.abi = &materialParams->program.abi;
        mateval(
            m.regs,
            materialParams->program.code,
            materialParams,
            ObjectToWorld,
            uint3(0),
            barys);

        return {
            m.EDFEval(wi),
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

// TODO: do not pass bsdf here, make the caller deal with it instead
// TODO: if we still want to pass bsdf, make this generic over BSDFs and make
// EvaluatedInterpretedMaterial implement it.
Spectrum sampleLd(
    bool vampire,
    float3 hitpos,
    float3 normal,
    float3 wo,
    __ref EvaluatedInterpretedMaterial bsdf,
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

            // TODO: normal should be geometric
            let Ld = fLight * bsdfEvaluation.f * (abs(dot(ls.wi, normal)) /
                (pdfLight + bsdfEvaluation.pdf));

            return Ld;
        }
    }

    return Spectrum(0);
}

// TODO: encapsulate this garbage into a nice Interaction struct or something
// like that.
bool closesthit(
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
    // TODO: rename to something like "vampireInstance" or whatever
    bool vampire = ((push->scene->accelInstances[instanceIndex].instanceIDAndMask >> 24) & 0b11) == 0b01;

    // TODO: rename to sampler or something
    var bn = ShiftedBlueNoise<R2LDS>(R2LDS(), dispatchRaysIndex.xy, 10 * path.depth);

    let materialParams = &push.scene->materialParams.data[instanceIndex * push.scene->maxPartsPerMesh + geometryIndex];

    // TODO: pass the entire thing from host, rather than constructing it here.
    IndexSlice idxBuffer = {(uint8_t*)materialParams->triangles, (uint64_t)2 << 62};

    let triIdxs = idxBuffer.Load3(primitiveIndex);

    // TODO: should not be called bsdf
    EvaluatedInterpretedMaterial bsdf;
    bsdf.abi = &materialParams->program.abi;
    mateval(
        bsdf.regs,
        materialParams->program.code,
        materialParams,
        ObjectToWorld,
        triIdxs,
        barys_yz);

    let wo = -WorldRayDirection;

    // TODO: this should be the eometric normal actually, i.e. perpendicular to
    // the triangle
    let normal = mul((float3x3)ObjectToWorld,
        interpolate3(
            float3x3(
                materialParams->normals[triIdxs[0]],
                materialParams->normals[triIdxs[1]],
                materialParams->normals[triIdxs[2]]),
            barys_yz));

    // Account for emission
    let Le = bsdf.EDFEval(wo);
    if (any(Le > float3(0))) {
        // TODO: this is wrong, we need to apply MIS weight, which we'll need to
        // consult lightSampler for, to figure out the pdf.
        path.L += path.beta * Le;
    }

    // TODO: we might want to sometimes write out the properties for a surface
    // that we intersect at a later bounce (e.g. reflection off a perfect
    // mirror, or transmission through a perfectly transmissive medium with
    // different IOR.) We'll want to think how to express that.
    if (path.depth == 0) {
        // TODO: export normal, depth, AOVs, etc, here

        // uint3 LaunchID = int3(DispatchRaysDimensions().x, 0, 0) + DispatchRaysIndex();

        // push.out.Store2D((int2)LaunchID.xy, float4(normal, 1));
    }

    path.depth++;

    if (path.depth >= push.camera->maxBounces)
        return false;

    // Direct light sampling
    {
        let u = bn.Get1D();
        let b = bn.Get2D();
        if (bsdf.Flags() & (BSDFFlags::Diffuse | BSDFFlags::Glossy)) {
            let Ld = sampleLd(vampire, hitpos, normal, wo, bsdf, u, b);

            path.L += path.beta * Ld;
        }
    }

    // Sample the bsdf for the next ray

    let bsdfSample = bsdf.Sample(wo, bn.Get1D(), bn.Get2D());

    // TODO: once we get diffuse cache, (optionally) terminate the path for
    // diffuse-only bsdfSamples and instead get the information from the cache.

    // TODO: transmission
    if (bsdfSample.flags & BSDFFlags::Reflected)
        path.beta *= bsdfSample.f * dot(bsdfSample.wi, normal) / bsdfSample.pdf;
    else
        return false;

    // Vampire bounces that are reflected become anti-vampire if the surface was
    // not vampire.
    //
    // Vampire bounces that are transmitted stay vampire.
    if (bsdfSample.flags & BSDFFlags::Reflected) {
        if (!vampire)
            path.vampire = false;
    }

    path.rayOrigin = hitpos; // TODO: this might come from bsdf as well
    path.rayDir = bsdfSample.wi; // TODO: should we normalize() here?

    // Russian roulette
    //
    // TODO: I guess we could also use an ordinary RNG here. Should we?
    float rr = bn.Get1D();
    if (path.depth >= push.camera->russianRouletteThreshold) {
        // TODO: introduce Spectrum.MaxComponentValue()
        float p = saturate(max(max(path.beta.x, path.beta.y), path.beta.z));
        if (!(rr < p))
            return false;
        path.beta /= p;
    }

    return true;
}
