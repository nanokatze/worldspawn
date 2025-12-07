import attribute;
import gpu;
import internal.integration.lds;
import internal.integration.mapping;
import internal.interpolation;
import internal.material.bsdf;
import internal.material.interpreter;
import internal.spectrum;
import light_triangle;
import lighting;
import scene;
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

// TODO: compartmentalize stuff into methods on Path?

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
            // lightAccel and apply the pdf (when lightAccel has a
            // possibility of returning the sky light.)
            path.L += path.beta * push.scene->lightAccel.env.SampleCubeLOD(float3(-V.y, V.z, V.x), 0).xyz;

            break;
        } else {
            /* abort(); */
            break;
        }
    }

    exportIrradiance(path.L);
}

struct ShaderRecordData {
    uint32_t *code;
};

[[vk::shader_record]] ConstantBuffer<ShaderRecordData> srd;

// TODO: do not pass bsdf here, make the caller deal with it instead
// TODO: if we still want to pass bsdf, make this generic over BSDFs and make
// EvaluatedInterpretedMaterial implement it.
Spectrum sampleDirectLight(
    bool vampire,
    float3 hitpos,
    float3 normal,
    float3 wo,
    __ref EvaluatedInterpretedMaterial bsdf,
    float asd,
    float2 uLight,
) {
    let lightAccel = &push.scene->lightAccel;

    // TODO: encapsulate this into lightAccel->SampleLight
    let emissiveInstanceIndex = (int)(asd * (float)lightAccel->emissiveInstanceCount);
    let emissiveInstance = &lightAccel->emissiveInstances.data[emissiveInstanceIndex];

    float u = fract(asd * (float)lightAccel->emissiveInstanceCount);

    let instancePMF = 1.0 / (float)lightAccel->emissiveInstanceCount;

    if (emissiveInstance->originalInstanceIndex != 0) {
        let lightMaterialParams = &push.scene->materialParams.data[
            emissiveInstance->originalInstanceIndex * push.scene->maxPartsPerMesh +
            emissiveInstance->originalGeometryIndex];

        IndexSlice idxs2 = {(uint8_t*)lightMaterialParams->meshPart.triangles, (uint64_t)2 << 62};

        let triPMF = 1.0 / (float)lightMaterialParams->meshPart.num_triangles;
        // TODO: we need to make sure this actually triangle < len always
        let triIdx = (int)((float)lightMaterialParams->meshPart.num_triangles * u);
        let vertIdxs = idxs2.Load3(triIdx);

        triangleLight light = {
            emissiveInstance->transform,
            vertIdxs,
            lightMaterialParams,
        };

        let d = light.SamplePoint(hitpos, uLight);

        RayDesc rd;
        rd.Origin = hitpos;
        rd.Direction = d.dir;
        rd.TMin = 0.0;
        // TODO: move this epsilon into a top level constant
        rd.TMax = max(d.dist - 0.001, 0);
        RayQuery<RAY_FLAG_FORCE_OPAQUE> rq;
        rq.TraceRayInline(
            rtas(push.scene->accel.ptr),
            0, // OR'd with flags above
            vampire ? 0b01 : 0b10,
            rd);
        rq.Proceed();

        if (rq.CommittedStatus() == COMMITTED_NOTHING) {
            let bsdfEvaluation = bsdf.Eval(wo, d.dir);
            let emissionSample = light.SampleEmission(hitpos, uLight);

            let emissionPDF = emissionSample.pdf * triPMF * instancePMF;

            let pdf = abs(dot(d.dir, normal)) / (bsdfEvaluation.pdf + emissionPDF);

            return bsdfEvaluation.f * emissionSample.f * pdf;
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

    var sampler = ShiftedBlueNoise<R2LDS>(R2LDS(), dispatchRaysIndex.xy, 10 * path.depth);

    let materialParams = &push.scene->materialParams.data[instanceIndex * push.scene->maxPartsPerMesh + geometryIndex];

    // TODO: pass the entire thing from host, rather than constructing it here.
    IndexSlice idxBuffer = {(uint8_t*)materialParams->meshPart.triangles, (uint64_t)2 << 62};

    let triIdxs = idxBuffer.Load3(primitiveIndex);

    let normal = mul((float3x3)ObjectToWorld,
        interpolate3(
            float3[3](
                materialParams->meshPart.normals[triIdxs[0]],
                materialParams->meshPart.normals[triIdxs[1]],
                materialParams->meshPart.normals[triIdxs[2]],
            ),
            barys_yz));

    EvaluatedInterpretedMaterial material;
    material.abi = &materialParams->program.abi;
    mateval(
        material.regs,
        materialParams->program.code,
        normal,
        (void*)&materialParams->args,
        ObjectToWorld,
        triIdxs,
        barys_yz);

    let wo = -WorldRayDirection;

    let emission = material.EDFEval(wo);
    if (any(emission > float3(0))) {
        // TODO: this is wrong, we need to apply MIS weight, which we'll need to
        // consult lightAccel for, to figure out the pdf.
        path.L += path.beta * emission;
    }

    // TODO: we might want to sometimes write out the properties for a surface
    // that we intersect at a later bounce (e.g. reflection off a perfect
    // mirror, or transmission through a perfectly transmissive medium with
    // different IOR.) We'll want to think how to express that.
    if (path.depth == 0) {
        // TODO: export normal, depth, AOVs, etc, here
        // TODO: the exported normal should be the normal averaged over all
        // BSDFs

        // uint3 LaunchID = int3(DispatchRaysDimensions().x, 0, 0) + DispatchRaysIndex();

        // push.out.Store2D((int2)LaunchID.xy, float4(normal, 1));
    }

    path.depth++;

    if (path.depth >= push.camera->maxBounces)
        return false;

    // Direct light sampling
    {
        let u = sampler.Get1D();
        let b = sampler.Get2D();
        if (material.Flags() & (BSDFFlags::Diffuse | BSDFFlags::Glossy)) {
            let L = sampleDirectLight(vampire, hitpos, normal, wo, material, u, b);

            path.L += path.beta * L;
        }
    }

    // Sample the BSDF for the next ray

    let bsdfSample = material.Sample(wo, sampler.Get1D(), sampler.Get2D());

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

    path.rayOrigin = hitpos; // TODO: this might come from material as well
    path.rayDir = bsdfSample.wi; // TODO: should we normalize() here?

    // Russian roulette
    //
    // TODO: I guess we could also use an ordinary RNG here. Should we?
    float rr = sampler.Get1D();
    if (path.depth >= push.camera->russianRouletteThreshold) {
        // TODO: introduce Spectrum.MaxComponentValue()
        float p = saturate(max(max(path.beta.x, path.beta.y), path.beta.z));
        if (!(rr < p))
            return false;
        path.beta /= p;
    }

    return true;
}
