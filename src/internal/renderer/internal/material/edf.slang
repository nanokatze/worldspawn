import spectrum;
import shading_frame;

public interface EDF {
    Spectrum Eval(float3 wi);
}

public struct EDFAdapter<T : EDF> : EDF {
    public Spectrum tint;
    public Frame shadingFrame; // TODO: shorter name?
    public T edf;

    public Spectrum Eval(float3 wi) {
        return tint * edf.Eval(shadingFrame.worldToLocal(wi));
    }
}

// TODO: move out of this file
// TODO: the shading frame should always be geometric normal, hmm
public struct UniformEDF : EDF {
    public Spectrum Eval(float3 wi) {
        return Spectrum(1); // wi.z != 0 ? f : (Spectrum)0;
    }
}
