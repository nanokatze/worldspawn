import spectrum;
import shading_frame;

// TODO: rename to smth like summary idk
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

public struct BSDFEval {
    public Spectrum f;
    public float pdf;

    public BSDFEval Add(BSDFEval other) {
        return {
            f + other.f,
            pdf + other.pdf,
        };
    }

    // public BSDFEval Scale(float scale) {
    //     return {
    //         f * scale,
    //         pdf * scale,
    //     };
    // }
};

public struct BSDFSample {
    public BSDFFlags flags;
    public Spectrum f;
    public float pdf;
    public float3 wi;
    // float eta;
};

public interface BSDF {
    BSDFFlags Flags();
    // TODO: albedo estimation
    // TODO: decouple spectrum evaluation from PDF evaluation?
    BSDFEval Eval(float3 wo, float3 wi);
    BSDFSample Sample(float3 wo, float uc, float2 u);
};

// TODO: rename
// This is basically the same as
// https://github.com/mmp/pbrt-v4/blob/f140d7cba5dc7b941f9346d6b7d1476a05c28c37/src/pbrt/bsdf.h
public struct BSDFAdapter<T : BSDF> : BSDF {
    public Spectrum tint;
    public Frame shadingFrame; // TODO: shorter name?
    public T bsdf; // TODO: rename to something like "underlying"

    BSDFFlags Flags() {
        // TODO: if tint is 0 or otherwise really tiny, we should just return 0
        return bsdf.Flags();
    }

    BSDFEval Eval(float3 wo, float3 wi) {
        let woLocal = shadingFrame.worldToLocal(wo);
        let wiLocal = shadingFrame.worldToLocal(wi);
        var e = bsdf.Eval(woLocal, wiLocal);
        e.f *= tint;
        return e;
    }

    BSDFSample Sample(float3 wo, float uc, float2 u) {
        let woLocal = shadingFrame.worldToLocal(wo);
        var sample = bsdf.Sample(woLocal, uc, u);
        sample.f *= tint;
        sample.wi = shadingFrame.localToWorld(sample.wi);
        return sample;
    }
}
