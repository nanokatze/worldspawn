import bsdf;
import spectrum;
import sampling;

static const float M_PI = 3.141592653589793;

// TODO: implement roughness

// TODO: rename to DiffuseBSDF?
public struct Diffuse : BSDF {
    public Spectrum R;
    // public float roughness;

    public BSDFFlags Flags() {
        // TODO: as an "opt" we should return 0 when R is 0
        return BSDFFlags::Diffuse | BSDFFlags::Reflected;
    }

    public BSDFEvaluatedResult Eval(float3 wo, float3 wi) {
        // TODO: rethink when we return zero
        if (wo.z < 1.5e-3f || wi.z < 1.5e-3f)
            return {};

        return {R / M_PI, CosineHemispherePDF(absCosTheta(wi))};
    }

    public BSDFSample Sample(float3 wo, float uc, float2 u) {
        float3 wi = SampleCosineHemisphere(u);

        return {
            R / M_PI,
            wi,
            CosineHemispherePDF(absCosTheta(wi)),
            Flags(),
        };
    }
};

float absCosTheta(float3 w) { return abs(w.z); }
