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
