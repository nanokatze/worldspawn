import bsdf;
import spectrum;
import sampling;

static const float M_PI_F = 3.141592653589793;

public struct Hehe {
    public Spectrum R;
    public Spectrum T;
    public float CosThetaT;
}

// TODO: provide at least a single impl
// TODO: rename to FresnelCurve or FC? Or FresnelModel?
public interface Fresnel {
    Hehe F();
}

float3 schlick(float3 f0, float cosTheta) {
  let k = 1.0f - cosTheta;
  let k2 = k * k;
  return f0 + (float3(1.0) - f0) * (k2 * k2 * k);
}

public struct TestFresnel : Fresnel {
    Hehe F() {
        return {
            Spectrum(1),
            Spectrum(0),
            1,
        };
    }
}

// TODO: we could start with perfectly smooth surfaces
// TODO: rename to MicrofacetDistribution?
public interface VNDF {
    float3 Sample(float3 wi, float2 alpha, float2 u);
}

float sqr(float x) { return x * x; }

float safe_sqrt(float x) { return sqrt(max(x, 0)); }

public struct GGX : VNDF {
    float3 Sample(float3 wi, float2 alpha, float2 u) {
        return {};
    }
}

public struct MicrofacetBSDF<D : VNDF, F : Fresnel> : BSDF {
    // public F fresnel;

    public BSDFFlags Flags() {
        return BSDFFlags::Reflected | BSDFFlags::Specular;
    }

    public BSDFEval Eval(float3 wo, float3 wi) {
        // TODO: rethink when we return zero
        if (wo.z < 1.5e-3f || wi.z < 1.5e-3f)
            return {};

        return {};
    }

    public BSDFSample Sample(float3 wo, float uc, float2 u) {
        // let fresnel_ss = schlick((float3)1, wo.z);

        return {
            BSDFFlags::Reflected | BSDFFlags::Specular,
            Spectrum(1) / abs(wo.z),
            1,
            float3(-wo.xy, wo.z),
        };
    }
}
