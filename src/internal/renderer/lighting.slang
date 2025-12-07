import internal.spectrum;

// Displacement from point on a surface to point on a light.
//
// TODO: questionable whether it really belong in this file...
public struct Displacement {
    public float3 dir;
    public float dist;
}

public struct EmissionSample {
    public Spectrum f;
    public float pdf; // TODO: make this the first field?
    public float3 normal; // TODO: why is this here?
}
