import bsdf;
import "../../spectrum";
import "../integration/mapping";

static const float M_PI = 3.141592653589793;

// TODO: implement roughness

public struct DiffuseBSDF : BSDF {
    // public float roughness;

    public BSDFFlags Flags() {
        return BSDFFlags::Diffuse | BSDFFlags::Reflected;
    }

    public BSDFEval Eval(float3 wo, float3 wi) {
        // TODO: rethink when we return zero
        if (wo.z < 1.5e-3f || wi.z < 1.5e-3f)
            return {};

        return {Spectrum(1) / M_PI, CosineHemispherePDF(absCosTheta(wi))};
    }

    public BSDFSample Sample(float3 wo, float uc, float2 u) {
        float3 wi = SampleCosineHemisphere(u);

        return {
            Flags(),
            Spectrum(1) / M_PI,
            CosineHemispherePDF(absCosTheta(wi)),
            wi,
        };
    }
};

float absCosTheta(float3 w) { return abs(w.z); }
