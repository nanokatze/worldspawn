
// This should be purely a description with no data associated.
//
// We should support everything we can get from blender.
struct AttributeDesc {
    int unused;
};

// TODO: we will sneak in the index type out of band so we'll construct it in
// the shader.
// TODO: see if we can replace this with AttributeDesc and attribute loads etc.
struct IndexSlice {
    uint8_t *data;
    uint64_t aux; // length in bytes and index type, packed

    uint32_t Load(int i) {
        int index_type = (int)(aux >> 62);

        if (index_type == 0)
           return i;

        uint64_t stride = 1 << (index_type - 1);
        uint64_t off = i * stride;

        // TODO: have reasonable robustness behavior, or abort on out of bounds
        // and do the checking on startup

        uint8_t *p = data + off;
        switch (index_type) {
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

    uint3 Load3(int i) {
        return uint3(Load(3 * i + 0), Load(3 * i + 1), Load(3 * i + 2));
    }
};
