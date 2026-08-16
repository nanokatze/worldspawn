#pragma once

typedef struct vec3 vec3;
struct vec3 {
	float x, y, z;
};

typedef struct dvec3 dvec3;
struct dvec3 {
	double x, y, z;
};

typedef struct mat4 mat4;
struct mat4 {
	float m[4][4];
};

typedef struct Rot3 Rot3;
struct Rot3 {
	float scalar, yz, zx, xy;
};
