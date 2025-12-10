[require(spirv, spirv_1_0)] module;

#define assert(x)

public interface LDS {
    float Get(int index, int dim);
}

// R_n sequence by M. Roberts, specialized for n=2
public struct R2LDS : LDS {
    float Get(int index, int dim) {
        constexpr let phi = generailizedGoldenRatio(2);
        return Rn(phi, index, dim);
    }
}

// n-dimensional R-sequence by M. Roberts
// https://extremelearning.com.au/unreasonable-effectiveness-of-quasirandom-sequences/
//
// TODO: implement https://www.martysmods.com/a-better-r2-sequence/
float Rn(float phi, int index, int dim) {
    assert(index >= 0);
    assert(dim >= 0);

    let alpha = pow(1.0 / phi, dim + 1);
    return fract(alpha * (index + 1));
}

// Solves x^(d+1) - x - 1 = 0 for positive x
float generailizedGoldenRatio(int d) {
    assert(d > 0);

    let iterations = 10;

    var x = (float)2.0;
    for (int i = 0; i < iterations; i++) {
        x = pow(1.0 + x, 1.0 / (float)(d + 1));
    }
    return x;
}

float mulMod1(float x, int y) {
    return mulMod1FixedPoint(x, y);
}

float mulMod1Naive(float x, int y) {
    return fract(x * (float)y);
}

float mulMod1FixedPoint(float x, int y) {
    assert(0 <= x);
    assert(0 <= y);

    let scale = 1ULL << 24;

    return (float)((uint32_t)(x * scale) * (uint32_t)y) / scale;
}
