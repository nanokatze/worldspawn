import gpu;
import spectrum;
import lighting;
import internal.material.interpreter;

// TODO: naming
public struct leSample {
    public Spectrum Le;
    public float3 normal;
    public float pdf;
}

// TODO: naming
public struct lightSampleA {
    public float3 wi;
    public float dist;
}

public struct emissiveInstance {
    public float3x4 transform;
    public uint32_t originalInstanceIndex;
    public uint32_t originalGeometryIndex;
};

// TODO: stable light IDs so that we can use them in RIS/ReSTIR etc
public struct lightSampler {
    public SamplingViewWithSampler env;

    public Slice<emissiveInstance> emissiveInstances;
    public uint64_t emissiveInstanceCount; // !
};

// TODO: should be moved to scene.slang
public struct MeshPart {
    public void *triangles;
    public uint32_t num_triangles;
    public float3 *posBuffer;
    public float3 *normals;
}

// TODO: kill kill kill
public struct MaterialParams {
    public interpreterProgram program;

    public MeshPart meshPart;

    // must be aligned to 8-byte boundary
    public uint8_t[256] args;
}

public struct Scene {
    public int64_t maxPartsPerMesh;

    public Slice<MaterialParams> materialParams;

    public Slice<AccelInstance> accelInstances;

    public Accel accel;

    public lightSampler lightSampler;

    public Sampler sampler;

    // there's more fields here, consult scene.go
};
