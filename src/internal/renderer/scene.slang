import gpu;

public struct MaterialParams {
    public uint32_t *code;
    public void *triangles;
    public float3 *normals;
    public float2 *uvs;
    public float hmm[3];
};

public struct Scene {
    public int64_t maxPartsPerMesh;

    public SamplingViewWithSampler sky;

    public Slice<MaterialParams> instances;
    public Slice<AccelInstance> accelInstances;
    public Accel accel;

    public Sampler sampler;

    // there's more fields here, consult scene.go
};
