import spectrum;

// TODO: rename to smth like BSDF sample properties
public enum BSDFSampleFlags {
    // Absorbed = 0,
    Emitted = 1 << 0,
    Reflected = 1 << 1,
    Transmitted = 1 << 2,
    Diffuse = 1 << 3,
    Glossy = 1 << 4,
    Specular = 1 << 5,
};

public struct BSDFSample {
    public Spectrum f;
    public float3 wi;
    public float pdf; // rename to just pd
    public BSDFSampleFlags flags;
    // float eta;
};

public struct BSDFEvaluatedResult {
    public Spectrum f;
    public float pdf; // rename to just pd
};

public interface BxDF {
    BSDFSample Sample(float3 wo, float uc, float2 u);
    BSDFEvaluatedResult Evaluate(float3 wo, float3 wi);
};
