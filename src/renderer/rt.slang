import gpu;
import camera;
import transfer_functions;

struct Push {
    Camera *camera;
    uint64_t tlas;
    StorageView out;
    SamplingViewWithSampler testTexture;
    // Mesh **meshInstances;
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
        rtas(push.tlas),
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
    rayPayload.color = push.camera->sky.SampleCubeLOD(float3(-rayDir.y, rayDir.z, rayDir.x), 0).xyz;
}

[shader("closesthit")]
void chit(inout RayPayload payload : SV_RayPayload, in float2 barys_yz : SV_IntersectionAttributes)
{
    // So, we'll have different materials be implemented by different shaders I
    // guess. Those different shaders will pull arguments from their specific
    // parameter structs. We probably should prefix the parameter struct with a
    // magic number so that we can diagnose bugs easily.

    float3 barys = (float3) {1.0 - barys_yz.x - barys_yz.y, barys_yz.x, barys_yz.y };

    /*
    Asdasd *asdasd = &push.asdasd[InstanceID()][GeometryIndex()];

    uint vi0 = 3 * PrimitiveIndex() + 0;
    uint vi1 = 3 * PrimitiveIndex() + 1;
    uint vi2 = 3 * PrimitiveIndex() + 2;

    float2 uv0 = asdasd->vertexUVs[vi0];
    float2 uv1 = asdasd->vertexUVs[vi1];
    float2 uv2 = asdasd->vertexUVs[vi2];
    float2 uv = mul(transpose(float3x2(uv0, uv1, uv2)), barys);
    */

    payload.color = push.testTexture.Sample2DLOD(barys.xy, 0).xyz;

    // MaterialTest *args = &push.materialTest[InstanceID()][GeometryIndex()];

    // TODO: use filtering after shading, in which case we'll be pulling texels rather than filtered samples.
    // payload.color = args->baseColorTexture.Sample2DLOD(uv, 0);
}

[shader("closesthit")]
void chit2(inout RayPayload payload : SV_RayPayload, in float2 barys_yz : SV_IntersectionAttributes)
{
    payload.color = (float3) {0.0, 1.0, 0.0};
}
