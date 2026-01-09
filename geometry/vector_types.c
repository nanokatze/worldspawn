#include "vector_types.h"

#include <math.h>

vec3
unpack_vec3(packed_vec3 packed)
{
	vec3 unpacked;
	unpacked.x = packed.x;
	unpacked.y = packed.y;
	unpacked.z = packed.z;
	return unpacked;
}

float dot3f(vec3 a, vec3 b) { return a.x*b.x + a.y*b.y + a.z*b.z; }

vec3
normalize3f(vec3 v)
{
	float norm = sqrtf(dot3f(v, v));
	if (norm > 0.0f) {
		v.x = v.x / norm;
		v.y = v.y / norm;
		v.z = v.z / norm;
	}
	return v;
}
