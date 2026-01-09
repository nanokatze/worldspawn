#pragma once

// TODO: just merge everything into the same header?

#include "vector_types.h"

// TODO: rename to mat4x4
typedef struct mat4 { float m[4][4]; } mat4;

vec4 matmul4x4x1(mat4 A, vec4 v);

mat4 matmul4x4x4(mat4 A, mat4 B);
