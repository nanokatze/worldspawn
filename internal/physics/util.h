#pragma once

#include <bit>
#include <array>

#include "gmath.h"

#include <Jolt/Jolt.h>

namespace worldspawn {

namespace physics {

inline vec3 JPHVec3ToVec3(JPH::Vec3 v)
{
	return vec3{v.GetX(), v.GetY(), v.GetZ()};
}

inline dvec3 JPHDVec3ToDVec3(JPH::DVec3 v)
{
	return dvec3{v.GetX(), v.GetY(), v.GetZ()};
}

inline JPH::Vec3 vec3ToJPHVec3(vec3 v)
{
	return JPH::Vec3(v.x, v.y, v.z);
}

inline JPH::DVec3 dvec3ToJPHDVec3(dvec3 v)
{
	return JPH::DVec3(v.x, v.y, v.z);
}

inline JPH::Quat rotation3ToJPHQuat(Rot3 r)
{
	return JPH::Quat(r.yz, r.zx, r.xy, r.scalar);
}

inline JPH::Mat44 mat4x4ToJPHMat44(mat4 m)
{
	// JPH::Mat44 are stored column-major and our matrices row major.
	return JPH::Mat44::sLoadFloat4x4(std::bit_cast<std::array<JPH::Float4, 4>>(m).data()).Transposed();
}

inline mat4 JPHMat44ToMat4x4(JPH::Mat44 m)
{
	std::array<JPH::Float4, 4> out;
	m.Transposed().StoreFloat4x4(out.data());
	return std::bit_cast<mat4>(out);
	// JPH::Mat44 are stored column-major and our matrices row major.
	// return JPH::Mat44::sLoadFloat4x4(std::bit_cast<std::array<JPH::Float4, 4>>(m).data()).Transposed();
}

}

}
