import attribute_buffer;
import film;
import gpu;
import internal.integration.lds;
import internal.integration.mapping;
import internal.interpolation;
import internal.material.bsdf;
import internal.material.interpreter;
import internal.spectrum;
import light;
import light_mesh_triangle;
import scene;
import transfer_functions;

#define SamplesPerBounce 10

// https://developer.nvidia.com/blog/rendering-in-real-time-with-spatiotemporal-blue-noise-textures-part-2/#getting_more_than_one_value_per_pixel
//
// TODO: can we somehow move this into internal/integration?
struct BlueNoiseSampler<T_lds : LDS> {
    T_lds lds; // TODO: move this elsewhere in the struct?
    int2 pixel;
    int sampleIndex;

    // TODO: most of our samples are 1D rather than 2D so we should make our
    // blue noise "multiplanar", i.e. R and G be in different channels
    [[mutating]]
    float Get1D() {
        return Get2D().x;
    }

    [[mutating]]
    float2 Get2D() {
        let sampleIndex = this.sampleIndex++;

        let point = float2(lds.Get(sampleIndex, 0), lds.Get(sampleIndex, 1));

        // TODO: make this a member on the BlueNoiseSampler or an interface or
        // idk.
        let blueNoise = push.blueNoise;
        let extent = blueNoise.Extent2D();

        let p = (pixel + int2(point * float2(extent))) & (extent - 1);

        // TODO: we could dither further down in-between samples. I guess we
        // could also use UINT to get the raw value and do format conversion
        // ourselves.
        return fract(blueNoise.Fetch2D(p).xy);
    }
}

struct Quality {
    int maxBounces;

    int russianRouletteThreshold;
};

struct cameraInternal {
    matrix<float, 4, 4> proj;
    matrix<float, 4, 4> view;

    matrix<float, 4, 4> viewProj;
    matrix<float, 4, 4> projInverse;
    matrix<float, 4, 4> viewInverse;

    void Ray(uint2 pixel, uint2 dimensions, __ref float3 rayOrigin, __ref float3 rayDir) {
        let pixelCenter = float2(pixel) + float2(0.5, 0.5);
        // TODO: the host should do the jittering
        let inUV = pixelCenter / float2(dimensions);
        let d = inUV * 2.0 - 1.0;
        let target = mul(push.camera.projInverse, float4(d.x, d.y, 1, 1));

        rayOrigin = mul(viewInverse, float4(0, 0, 0, 1)).xyz;
        // TODO: do we actually need normalize here?
        rayDir = normalize(mul(viewInverse, float4(normalize(target.xyz), 0)).xyz);
    }
}

struct frameParams {
    Scene *scene;

    uint32_t number;

    ImageDescriptors blueNoise;

    cameraInternal camera;
    Film film;

    Quality quality;
}

[[vk::push_constant]] frameParams *push;

RaytracingAccelerationStructure rtas(uint64_t bda) {
    return spirv_asm { result:$$RaytracingAccelerationStructure = OpConvertUToAccelerationStructureKHR $bda };
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

// TODO: compartmentalize stuff into methods on Path?

[[shader("raygeneration")]]
void raygen() {
    let LaunchID = DispatchRaysIndex().xy;
    let LaunchSize = DispatchRaysDimensions().xy;

    Path path = {};
    path.beta = Spectrum(1.0);
    path.vampire = true;
    push.camera.Ray(LaunchID, LaunchSize, path.rayOrigin, path.rayDir);

    for (;;) {
        RayQuery<RAY_FLAG_FORCE_OPAQUE> rq;
        rq.TraceRayInline(
            rtas(push.scene->accel.ptr),
            0, // OR'd with flags above
            path.vampire ? 0b01 : 0b10,
            RayDesc(path.rayOrigin, 0, path.rayDir, 1.0 / 0.0));
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

            let V = rq.WorldRayDirection();
            let W = mul(push.scene->lightAccel.envTransform, V);

            // TODO: use SampleCubeGrad. We can use aniso sampling for cube textures.
            // Actually, aniso sampling is broken across many impls. We might wanna look
            // into filtering after shading for sky as well, but hopefully things might
            // get fixed.
            // TODO: same as with emissive surfaces/volumes, we need to consult
            // lightAccel and apply the pdf (when lightAccel has a
            // possibility of returning the sky light.)
            // TODO: pass the matrix for converting V. Call it skyTransform or something

            path.L += path.beta * push.scene->lightAccel.env.SampleCubeLOD(push.scene->sampler, W, 0).xyz;

            break;
        } else {
            /* abort(); */
            break;
        }
    }

    push.film.color.Store2D(LaunchID, float4(oetf(0, path.L), 1));
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

    let u = fract(asd * (float)lightAccel->emissiveInstanceCount);

    let instancePMF = 1.0 / (float)lightAccel->emissiveInstanceCount;

    if (emissiveInstance->originalInstanceIndex != 0) {
        let lightMaterialParams = &push.scene->materialParams.data[
            emissiveInstance->originalInstanceIndex * push.scene->maxPartsPerMesh +
            emissiveInstance->originalGeometryIndex];

        IndexBuffer idxs2 = lightMaterialParams->meshPart.indexBuffer;

        let triPMF = 1.0 / (float)(idxs2.Len() / 3);
        // TODO: we need to make sure this actually triangle < len always
        let triIdx = (int)((float)(idxs2.Len() / 3) * u);
        let vertIdxs = idxs2.Load3(triIdx);

        meshTriangleLight light = {
            emissiveInstance->transform,
            vertIdxs,
            lightMaterialParams,
        };

        let d = light.SamplePoint(hitpos, uLight);

        RayQuery<RAY_FLAG_FORCE_OPAQUE> rq;
        rq.TraceRayInline(
            rtas(push.scene->accel.ptr),
            0, // OR'd with flags above
            vampire ? 0b01 : 0b10,
            // TODO: move the epsilon into a top level constant
            RayDesc(hitpos, 0, d.dir, max(d.dist - 0.001, 0)));
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

// TODO: expand this structure and use it in more situations
public struct Interaction {
    public float2 barys_yz;

    // public uint primitiveIndex;
    public uint3 vertexIndices;

    float4 Attribute(AttributeBuffer buffer) {
        return interpolate4(
            float4[3](
                buffer.Load(vertexIndices[0]),
                buffer.Load(vertexIndices[1]),
                buffer.Load(vertexIndices[2]),
            ),
            barys_yz);
    }
}

// TODO: encapsulate this garbage into a nice Interaction struct or something
// like that.
bool closesthit(
    __ref Path path,
    uint2 dispatchRaysIndex,
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
    let vampire = ((push.scene->accelInstances[instanceIndex].instanceIDAndMask >> 24) & 0b11) == 0b01;

    BlueNoiseSampler<R2LDS> sampler = {
        R2LDS(),
        dispatchRaysIndex,
        push.number * 69 * SamplesPerBounce + path.depth * SamplesPerBounce,
    };

    let materialParams = &push.scene->materialParams.data[instanceIndex * push.scene->maxPartsPerMesh + geometryIndex];

    IndexBuffer idxBuffer = materialParams->meshPart.indexBuffer;

    Interaction interaction = {
        barys_yz,
        idxBuffer.Load3(primitiveIndex),
    };

    AttributeBuffer normalBuffer = {
        0,
        (uint8_t*)materialParams->meshPart.normals,
        0,
        12,
    };

    let normal = mul((float3x3)ObjectToWorld, interaction.Attribute(normalBuffer).xyz);

    EvaluatedInterpretedMaterial material;
    material.abi = &materialParams->program.abi;
    mateval(
        material.regs,
        materialParams->program.code,
        normal,
        (void*)&materialParams->args,
        ObjectToWorld,
        interaction.vertexIndices,
        barys_yz);

    // TODO: when we get around to using separate shaders, specialize them for
    // primary and non-primary bounces?
    // TODO: implement "primary surface replacement"
    if (path.depth == 0) {
        // TODO: proper diffuse and specular albedos
        push.film.diffuseAlbedo.Store2D(dispatchRaysIndex, float4(material.Albedo(), 1));
        // TODO: transform normal properly
        // TODO: spell out that it's intended to be denoising normal?
        // TODO: drop one of the components? We can also do octahexdral encoding
        // but it's not clear whether we really need it (in full generality)
        // given that one of the components should always be positive.
        // TODO: roughness
        push.film.normalAndRoughness.Store2D(dispatchRaysIndex, float4(mul(float3x3(push.camera.view), normal), 1));

        let hitposClip = mul(push.camera.viewProj, float4(hitpos, 1));
        let depth = hitposClip.z / hitposClip.w;
        push.film.depth.Store2D(dispatchRaysIndex, float4(depth, 0, 0, 1));
    }

    let wo = -WorldRayDirection;

    let emission = material.EDFEval(wo);
    if (any(emission > float3(0))) {
        // TODO: this is wrong, we need to apply MIS weight, which we'll need to
        // consult lightAccel for, to figure out the pdf.
        path.L += path.beta * emission;
    }

    path.depth++;

    if (path.depth >= push.quality.maxBounces)
        return false;

    // Direct light sampling
    {
        let u = sampler.Get1D();
        let b = sampler.Get2D();
        if (material.Flags() & (BSDFFlags::Diffuse | BSDFFlags::Glossy)) {
            let L = sampleDirectLight(path.vampire && vampire, hitpos, normal, wo, material, u, b);

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
    let rr = sampler.Get1D();
    if (path.depth >= push.quality.russianRouletteThreshold) {
        // TODO: introduce Spectrum.MaxComponentValue()
        float p = saturate(max(max(path.beta.x, path.beta.y), path.beta.z));
        if (!(rr < p))
            return false;
        path.beta /= p;
    }

    return true;
}
