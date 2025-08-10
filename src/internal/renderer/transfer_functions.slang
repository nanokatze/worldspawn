module otf;

// See
// https://registry.khronos.org/DataFormat/specs/1.3/dataformat.1.3.html#TRANSFER_CONVERSION

// TODO: make these methods on objects implementing some interface?

public float3 oetf(int colorSpace, float3 RGB) {
    return float3(
        oetf2(colorSpace, RGB.x),
        oetf2(colorSpace, RGB.y),
        oetf2(colorSpace, RGB.z));
}

float oetf2(int colorSpace, float L) {
    switch (colorSpace) {
    case 0: {
        if (L <= 0.0031308)
            return 12.92 * L;
        return 1.055 * pow(L, 1.0 / 2.4) - 0.055;
    }
    }

    // assert(0);
    return L;
}
