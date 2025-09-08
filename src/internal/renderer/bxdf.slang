import spectrum;

// TODO: rename to smth like surface interaction or idk
public enum BSDFFlags {
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
    public float pdf;
    public BSDFFlags flags;
    // float eta;
};

// TODO: rename
public struct BSDFEvaluatedResult {
    public Spectrum f;
    public float pdf;
};

public interface BxDF {
    BSDFFlags Flags();
    // Spectrum F(float3 wo, float3 wi);
    // TODO: decouple spectrum evaluation from PDF evaluation
    BSDFEvaluatedResult Eval(float3 wo, float3 wi);
    BSDFSample Sample(float3 wo, float uc, float2 u);
};
