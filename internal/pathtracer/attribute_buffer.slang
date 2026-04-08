public struct AttributeBuffer {
    public uint32_t format;
    public uint8_t *data;
    public uint64_t len;    // in bytes
    public uint64_t stride; // in bytes

    public float4 Load(uint32_t i) {
        let p = data + i * stride;

        var result = float4(0, 0, 0, 1);
        // TODO: this should probably be a custom format list so that we could
        // add octahedrally encoded formats and stuff
        switch (format) {
        case 0:
            result.xyz = *(float3*)p;
            break;
        default:
            break;
        }
        return result;
    }
}
