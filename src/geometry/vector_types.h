#pragma once

#include <stdint.h>

#define DEFINE_VEC2_TYPE(elem, name) \
	typedef struct name { elem x, y; } name
#define DEFINE_VEC3_TYPE(elem, name) \
	typedef struct name { elem x, y, z; } name
#define DEFINE_VEC4_TYPE(elem, name) \
	typedef struct name { elem x, y, z, w; } name

DEFINE_VEC2_TYPE(int32_t, int32x2);

DEFINE_VEC4_TYPE(uint32_t, uint32x4);

DEFINE_VEC2_TYPE(float, vec2);
DEFINE_VEC3_TYPE(float, vec3);
DEFINE_VEC4_TYPE(float, vec4);

DEFINE_VEC3_TYPE(double, dvec3);

#undef DEFINE_VEC2_TYPE
#undef DEFINE_VEC3_TYPE
#undef DEFINE_VEC4_TYPE

typedef struct packed_vec3 {
	float x, y, z;
} packed_vec3;

vec3 unpack_vec3(packed_vec3 packed);

float dot3f(vec3 a, vec3 b);

vec3 normalize3f(vec3 v);

// vec3 lerp3f(vec3 a, vec3 b, float c);
