public struct IndexBuffer {
    public uint8_t indexType;
    public uint8_t *data;
    public uint64_t len; // in elements

    // TODO: constructor that takes Slice<byte> and indexType

    // TODO: factor things out into bounds check and load without bounds
    // checking? If e.g. Load3 even partially ends up out of bounds, we'd return
    // 0xffffffff in the entire uint3
    // TODO: use int32 instead of uint32?

    public uint32_t Len() {
        return (uint32_t)len;
    }

    public uint32_t Load(uint32_t i) {
        if (indexType == 0)
            return i;

        let stride = 1 << (indexType - 1);
        let off = i * stride;

        let p = data + off;
        switch (indexType) {
        case 1:
            return *(uint8_t*)p;
        case 2:
            return *(uint16_t*)p;
        case 3:
            return *(uint32_t*)p;
        default:
            // we should be able to put unreachable() here
            return i;
        }
    }

    public uint2 Load2(uint32_t i) {
        let indices = uint2(i) + uint2(0, 1);
        return uint2(Load(indices[0]), Load(indices[1]));
    }

    public uint3 Load3(uint32_t i) {
        let indices = uint3(i) + uint3(0, 1, 2);
        return uint3(Load(indices[0]), Load(indices[1]), Load(indices[2]));
    }
}
