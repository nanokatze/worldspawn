[require(spirv, spirv_1_0)] module;

#define assert(x)

public interface LDS {
	public float Get(int dimension, int i);
}

// n-dimensional R-sequence by M. Roberts
// https://extremelearning.com.au/unreasonable-effectiveness-of-quasirandom-sequences/
//
// TODO: see also https://www.martysmods.com/a-better-r2-sequence/
public struct Roberts : LDS {
	float phi_d;

	public __init(int d) {
		phi_d = _RobertsPhi(d);
	}

	// TODO: swap dimension and point index
	[require(spirv, spirv_1_0)]
	public float Get(int dimension, int i) {
		assert(dimension >= 0);
		assert(i >= 0);
		float alpha = pow(1.0 / phi_d, dimension + 1);
		return mulMod1(alpha, i + 1);
	}
}

// TODO: rename
float _RobertsPhi(int d) {
	assert(d > 0);

	int iterations = 10;

	float phi_d = 2.0;
	for (int i = 0; i < iterations; i++) {
		phi_d = pow(1.0 + phi_d, 1.0 / (float)(d + 1));
	}
	return phi_d;
}

float mulMod1(float x, int y) {
	return mulMod1FixedPoint(x, y);
}

float mulMod1Naive(float x, int y) {
	return fmod(x * (float)y, 1);
}

float mulMod1FixedPoint(float x, int y) {
	assert(0 <= x);
	assert(0 <= y);

	float scale = 1ULL << 24;

	return (float)((uint32_t)(x * scale) * (uint32_t)y) / scale;
}
