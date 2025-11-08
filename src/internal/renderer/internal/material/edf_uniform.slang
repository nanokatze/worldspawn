import edf;
import "../../spectrum";

public struct UniformEDF : EDF {
    public Spectrum Eval(float3 wi) {
        return Spectrum(1); // wi.z != 0 ? f : (Spectrum)0;
    }
}
