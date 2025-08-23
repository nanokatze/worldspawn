import bsdf;
import spectrum;
import sampling;

static const float M_PI = 3.141592653589793;

public struct Diffuse : BxDF {
    public Spectrum R;
    public float roughness;

    public BSDFSample Sample(float3 wo, float uc, float2 u) {
        float3 wi = SampleCosineHemisphere(u);

        return {
            R / M_PI,
            wi,
            CosineHemispherePDF(absCosTheta(wi)),
            BSDFSampleFlags::Reflected,
        };
    }

    public BSDFEvaluatedResult Evaluate(float3 wo, float3 wi) {
        if (wo.z < 1.5e-3f || wi.z < 1.5e-3f)
            return {};

        return {R / M_PI, CosineHemispherePDF(absCosTheta(wi))};
    }
};

// public struct

float absCosTheta(float3 w) { return abs(w.z); }
