import bsdf;
import spectrum;
import sampling;

static const float M_PI_F = 3.141592653589793;

// TODO: provide at least a single impl
public interface Fresnel {
    // void F()
}

// TODO: we could start with perfectly smooth surfaces
public interface NDF {
    // TODO: we need a family of sample functions basically
    float3 Sample();
}

public struct MicrofacetBSDF<F : Fresnel, D : NDF> : BSDF {
    // public F fresnel;

    public BSDFFlags Flags() {
        return BSDFFlags::Reflected;
    }

    public BSDFEvaluatedResult Eval(float3 wo, float3 wi) {
        // TODO: rethink when we return zero
        if (wo.z < 1.5e-3f || wi.z < 1.5e-3f)
            return {};

        return {};
    }

    public BSDFSample Sample(float3 wo, float uc, float2 u) {
        return {};
    }
}
