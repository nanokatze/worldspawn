import attribute_buffer;
import gpu;
import internal.material.interpreter;
import internal.spectrum;

public struct emissiveInstance {
    public float3x4 transform;
    public uint32_t originalInstanceIndex;
    public uint32_t originalGeometryIndex;
};

// TODO: stable light IDs so that we can use them in RIS/ReSTIR etc
// TODO: rename to lightAccel
public struct lightAccel {
    public float3x3 envTransform;
    public ImageDescriptors env;

    public Slice<emissiveInstance> emissiveInstances;
    public uint64_t emissiveInstanceCount; // !
};

public struct MeshPart {
    public IndexBuffer indexBuffer;
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

    public lightAccel lightAccel;

    public uint32_t sampler;

    // there's more fields here, consult scene.go
};
