import gpu;

struct gfxParams {
    float4x4 projInv;
    float4x4 viewInv;
    Accel tlas;
    ImageDescriptor output;
}

[[vk::push_constant]] gfxParams gfxPush;

[[shader("raygeneration")]]
void gfxView() {
    let LaunchID = DispatchRaysIndex().xy;
    let LaunchSize = DispatchRaysDimensions().xy;

    let pixelCenter = float2(LaunchID) + float2(0.5, 0.5);
    let inUV = pixelCenter / float2(LaunchSize);
    let d = inUV * 2.0 - 1.0;
    let target = mul(gfxPush.projInv, float4(d.x, d.y, 1, 1));

    let rayOrigin = mul(gfxPush.viewInv, float4(0, 0, 0, 1)).xyz;
    let rayDir = normalize(mul(gfxPush.viewInv, float4(normalize(target.xyz), 0)).xyz);

    RayQuery<RAY_FLAG_FORCE_OPAQUE> rq;
    rq.TraceRayInline(
        rtas(gfxPush.tlas.ptr),
        0, // OR'd with flags above
        0x01,
        RayDesc(rayOrigin, 0, rayDir, 1024));
    rq.Proceed();

    var color = float3(0.6, 0.8, 1);
    if (rq.CommittedStatus() == COMMITTED_TRIANGLE_HIT) {
        let barys_yz = rq.CommittedTriangleBarycentrics();
        color = float3(1.0 - barys_yz.x - barys_yz.y, barys_yz.x, barys_yz.y);
    }

    gfxPush.output.Store2D(LaunchID, float4(color, 1));
}

struct Vertex {
    uint32_t emitter;
    float t;
    float attenuation; // attenuation TODO: we should also
}

struct Path {
    float t; // traveled time
    float throughput;
    float propagationVelocity;
    float3 rayOrigin; // we do kinda need this ig for microphones (teleporters basically)
    float3 rayDir;
    // Path vertices excluding the ear vertex
    Slice<Vertex> vertices;
}

struct traceParams {
    Accel tlas;
    // TODO: specify max bounces and russian roulette
    // TODO: later we should probs replace this with InitialState object which
    // would be responsible for setting up Path
    Path *pathStates;
}

[[vk::push_constant]] traceParams tracePush;

struct Ears {
    // void Ray(uint point, uint )
}

struct Emitter {
    uint32_t id;
    float3 pos;
}

Slice<T> append<T>(Slice<T> s, T v) {
    if (s.len >= s.cap)
        return s; // TODO: abort() in this case

    uint64_t i = s.len;
    s.data[i] = v;
    s.len++;

    return s;
}

[[shader("raygeneration")]]
void trace() {
    let LaunchID = DispatchRaysIndex().x;

    // TODO: replace this with InitialState object or smth similar (we could
    // call it Ears or Head lmao) which would be responsible for setting up
    // Path. We should probably do the same in pathtracer/scene_render.slang,
    // there it really feels like a responsibility of the Camera object

    let path = &tracePush.pathStates[LaunchID];

    for (int i = 0; i < 2; i++) {
        {
            let hitpos = path->rayOrigin;

            let time = path->t;

            let throughput = path->throughput;

            Emitter emitter = {
                42,
                float3(0, 0, 0),
            };

            let d = emitter.pos - hitpos;

            RayQuery<RAY_FLAG_FORCE_OPAQUE> rq;
            rq.TraceRayInline(
                rtas(tracePush.tlas.ptr),
                0, // OR'd with flags above
                0x01,
                RayDesc(hitpos, 0, normalize(d), max(length(d) - 0.001, 0)));
            rq.Proceed();

            if (rq.CommittedStatus() == COMMITTED_NOTHING) {
                Vertex vtx = {
                    emitter.id,
                    time + rq.CommittedRayT() / path->propagationVelocity,
                    throughput,
                };

                path->vertices = append(path->vertices, vtx);
            }
        }

        RayQuery<RAY_FLAG_FORCE_OPAQUE> rq;
        rq.TraceRayInline(
            rtas(tracePush.tlas.ptr),
            0, // OR'd with flags above
            0x01,
            RayDesc(path->rayOrigin, 0, path->rayDir, 1024));
        rq.Proceed();

        if (rq.CommittedStatus() == COMMITTED_TRIANGLE_HIT) {
            let hitpos = rq.WorldRayOrigin() + rq.WorldRayDirection() * max(rq.CommittedRayT() - 0.001, 0);

            let time = path->t + rq.CommittedRayT() / path->propagationVelocity;

            let throughput = path->throughput * 0.69;

            path->rayOrigin = hitpos;
            path->t = time;
            path->throughput = throughput;
        } else {
            return;
        }
    }
}

// TODO: see if there's something we can replace stuff with
RaytracingAccelerationStructure rtas(uint64_t bda) {
    return spirv_asm { result:$$RaytracingAccelerationStructure = OpConvertUToAccelerationStructureKHR $bda };
}
