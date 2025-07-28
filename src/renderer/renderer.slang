import gpu;
import transfer_functions;

struct Mesh {
    uint16_t *triangles;
    float2 *uvs;
    SamplingViewWithSampler testTexture;
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

[[vk::push_constant]] Push push;

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

[shader("closesthit")]
void chit(inout RayPayload payload : SV_RayPayload, in float2 barys_yz : SV_IntersectionAttributes)
{
    float3 barys = float3(1.0 - barys_yz.x - barys_yz.y, barys_yz);

    Mesh *instance = push.scene->instances[InstanceID()];

    uint16_t vi0 = instance->triangles[3 * PrimitiveIndex() + 0];
    uint16_t vi1 = instance->triangles[3 * PrimitiveIndex() + 1];
    uint16_t vi2 = instance->triangles[3 * PrimitiveIndex() + 2];

    float2 uv0 = instance->uvs[vi0];
    float2 uv1 = instance->uvs[vi1];
    float2 uv2 = instance->uvs[vi2];
    // TODO: make a dedicated interpolation function which will take care of the
    // edge cases
    float2 uv = mul(transpose(float3x2(uv0, uv1, uv2)), barys);

    // TODO: not sure we should be flipping the UV here. Maybe we should do that
    // when exporting the textures?
    payload.color = instance->testTexture.Sample2DLOD(float2(uv.x, 1-uv.y), 0).xyz;
}

[shader("closesthit")]
void chit2(inout RayPayload payload : SV_RayPayload, in float2 barys_yz : SV_IntersectionAttributes)
{
    payload.color = (float3) {0.0, 1.0, 0.0};
}
