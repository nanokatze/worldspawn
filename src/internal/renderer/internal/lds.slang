[require(spirv, spirv_1_0)] module;

#define assert(x)

public interface LDS {
	[[mutating]]
	public float Get();

	// [[mutating]]
	// public float Get2D();
}

// n-dimensional R-sequence by M. Roberts
// https://extremelearning.com.au/unreasonable-effectiveness-of-quasirandom-sequences/
//
// TODO: see also https://www.martysmods.com/a-better-r2-sequence/
// TODO: copy rtx remix's getNextSampleBlueNoise.
public struct Roberts : LDS {
	float phi_d;
	int point;
	int dim;

	public __init(int d, int pt = 0, int startingdim = 0) {
		phi_d = _RobertsPhi(d);
		point = pt;
		dim = startingdim;
	}

	// TODO: rename to Get1D and introduce Get2D for proper 2D values (two
	// consecutive 1D samples are correlated)
	[[mutating]]
	public float Get() {
		return Get(dim++, point);
	}

	// TODO: make this private
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
