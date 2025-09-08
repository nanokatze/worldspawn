import gpu;
import spectrum;
import sampling;
import lighting;

public struct MaterialParams {
    public uint32_t *code;
    public void *triangles;
    public float3 *normals;
    public float2 *uvs;
    public float hmm[3];
    public float emission[3];
};

float3
interpolate3(float3x3 attrs, float2 barys)
{
    return mul(transpose(attrs), float3(1.0 - barys.x - barys.y, barys));
}

// TODO: instead of having public light objects, we should instead pass light
// identifier, which could be e.g. an uint32. This light identifier would be
// used by RIS/ReSTIR, and would have to be passed back to lightSampler in order
// to sample the light.

public struct light {
    // TODO: lights should just point to triangles (or quads) in meshes.
    public float3 verts[3];
    public Spectrum emission;

    // TODO: rename
    public Spectrum Li(float2 idk) {
        return emission;
    }

    // TODO: rename hitpos to something else, like p_surf or w/e
    public lightSample Sample(float3 hitpos, float2 idk) {
        // TODO: switch to spherical triangle sampling for better distribution
        // instead.

        let area = length(cross(verts[1] - verts[0], verts[2] - verts[0])) * 0.5;
        let normal = normalize(cross(verts[1] - verts[0], verts[2] - verts[0]));

        let barys = SampleTriangle(idk);
        let p = interpolate3(reinterpret<float3x3>(verts), barys);

        let uhh = p - hitpos;

        let wi = normalize(uhh);

        return {
            wi,
            length(uhh),
            normal,
            dot(uhh, uhh) / (abs(dot(normal, wi)) * area),
        };
    }
};

public struct emissiveInstance {
    public float3x4 transform;
    public Slice<float3> posBuffer;
    public Slice<uint16_t3> indexBuffer;
};

public struct lightSampler {
    public SamplingViewWithSampler env;

    public Slice<emissiveInstance> emissiveInstances;
    public uint64_t emissiveInstanceCount; // !
};

public struct Scene {
    public int64_t maxPartsPerMesh;

    public Slice<MaterialParams> materialParams;

    public Slice<AccelInstance> accelInstances;

    public Accel accel;

    public lightSampler lightSampler;

    public Sampler sampler;

    // there's more fields here, consult scene.go
};
