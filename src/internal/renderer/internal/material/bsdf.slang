import spectrum;

// TODO: rename this file to bsdf.slang

// TODO: rename to smth like surface interaction or idk
// TODO: move out of this file
// TODO: combine flags so that we have ReflectionDiffuse, ReflectionGlossy, etc?
public enum BSDFFlags {
    // Absorbed = 0,
    Emitted = 1 << 0,
    Reflected = 1 << 1,
    Transmitted = 1 << 2,
    Diffuse = 1 << 3,
    Glossy = 1 << 4,
    Specular = 1 << 5,
};

// TODO: shorter name, e.g. BSDFEval?
public struct BSDFEvaluatedResult {
    public Spectrum f;
    public float pdf;

    public BSDFEvaluatedResult Add(BSDFEvaluatedResult other) {
        return {
            f + other.f,
            pdf + other.pdf,
        };
    }

    // public BSDFEvaluatedResult Scale(float scale) {
    //     return {
    //         f * scale,
    //         pdf * scale,
    //     };
    // }
};

public struct BSDFSample {
    public Spectrum f;
    public float3 wi;
    public float pdf;
    public BSDFFlags flags;
    // float eta;
};

// rename to BSDF
public interface BSDF {
    BSDFFlags Flags();
    // TODO: albedo estimation
    // TODO: decouple spectrum evaluation from PDF evaluation?
    BSDFEvaluatedResult Eval(float3 wo, float3 wi);
    BSDFSample Sample(float3 wo, float uc, float2 u);
};

// TODO: replace this with just a float3x3/Mat3x3
public struct Frame {
    public float3 x, y, z;

    public static Frame fromNormal(float3 n) {
        float3 a = abs(n.x) > 0.5 ? float3(0, 0, 1) : float3(1, 0, 0);

        float3 b = normalize(cross(n, a));
        float3 t = cross(n, b);

        return {t, b, n};
    }

    public static Frame fromNT(float3 n, float3 t, float sign = 1.0) {
        if (abs(dot(n, t)) > 0.9) return fromNormal(n);

        float3 b = normalize(cross(n, t)) * sign;
        t = cross(b, n);

        return {t, b, n};
    }

    public float3 worldToLocal(float3 w) {
        return float3(dot(w, x), dot(w, y), dot(w, z));
    }

    public float3 localToWorld(float3 l) {
        return x * l.x + y * l.y + z * l.z;
    }
};

// TODO: rename
// This is basically the same as
// https://github.com/mmp/pbrt-v4/blob/f140d7cba5dc7b941f9346d6b7d1476a05c28c37/src/pbrt/bsdf.h
public struct BSDFAdapter<T : BSDF> : BSDF {
    // TODO: also include color scale here
    public Frame shadingFrame;
    public T bsdf;

    BSDFFlags Flags() {
        return bsdf.Flags();
    }

    BSDFEvaluatedResult Eval(float3 wo, float3 wi) {
        let wo2 = shadingFrame.worldToLocal(wo);
        let wi2 = shadingFrame.worldToLocal(wi);
        return bsdf.Eval(wo2, wi2);
    }

    BSDFSample Sample(float3 wo, float uc, float2 u) {
        let wo2 = shadingFrame.worldToLocal(wo);
        BSDFSample sample = bsdf.Sample(wo2, uc, u);
        return {
            sample.f,
            shadingFrame.localToWorld(sample.wi),
            sample.pdf,
            sample.flags,
        };
    }
}

// TODO: move out of this file
public struct UniformEDF {
    public Spectrum f;

    public Spectrum Eval(float3 wo) {
        return f;
    }
}
