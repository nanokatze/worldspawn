public struct lightSample {
    // Direction from the surface point to the light source point.
    public float3 wi;

    // Distance from the surface point to the light source point. Infinity for
    // infinite light sources.
    public float dist;

    // Normal at the light source point.
    public float3 normal;

    public float pdf;
};
