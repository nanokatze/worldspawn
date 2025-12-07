import internal.integration.mapping;
import internal.interpolation;
import internal.material.interpreter;
import lighting;
import scene; // TODO: stop importing this

// TODO: this to mesh triangle light? To make it clear that it's backed by a
// mesh.

// TODO: clean up.
public struct triangleLight {
    public float3x4 ObjectToWorld;

    public uint3 indices;

    public MaterialParams *materialParams;

    float3[3] verts() {
        return float3[3](
            mul(ObjectToWorld, float4(materialParams->meshPart.posBuffer[indices[0]], 1.0)),
            mul(ObjectToWorld, float4(materialParams->meshPart.posBuffer[indices[1]], 1.0)),
            mul(ObjectToWorld, float4(materialParams->meshPart.posBuffer[indices[2]], 1.0)),
        );
    }

    // TODO: switch to spherical triangle sampling for better distribution
    // instead.

    // Sample a point on the light.
    //
    // TODO: rename hitpos or pass Intersection or idk
    public Displacement SamplePoint(float3 hitpos, float2 u) {
        let verts = this.verts();
        let barys = SampleTriangle(u);
        let p = interpolate3(verts, barys);
        return {normalize(p - hitpos), length(p - hitpos)};
    }

    // Light emitted from a sampled point. u must be the same as what was used
    // for sampling the point.
    public EmissionSample SampleEmission(float3 hitpos, float2 u) {
        let barys = SampleTriangle(u);

        let verts = this.verts();

        let area = length(cross(verts[1] - verts[0], verts[2] - verts[0])) * 0.5;
        // TODO: should this really be geometric normal?
        let normal = normalize(cross(verts[1] - verts[0], verts[2] - verts[0]));

        let p = interpolate3(verts, barys);

        let uhh = p - hitpos; // d?

        let wi = normalize(uhh); // dir

        EvaluatedInterpretedMaterial material;
        material.abi = &materialParams->program.abi;
        mateval(
            material.regs,
            materialParams->program.code,
            float3(0),
            (void*)&materialParams->args,
            ObjectToWorld,
            indices,
            barys);

        return {
            material.EDFEval(wi),
            dot(uhh, uhh) / (abs(dot(normal, wi)) * area),
            normal,
        };
    }
};
