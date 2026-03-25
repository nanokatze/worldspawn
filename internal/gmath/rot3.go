package gmath

import (
	"math"
)

// TODO: kill quat and generate RotN types

type Rot3 [4]float32

func Rot3One() Rot3 {
	return Rot3{0, 0, 0, 1}
}

func Rot3InPlane(plane Vec3, θ float32) Rot3 {
	s, c := math.Sincos(float64(θ / 2))
	yz := plane[0] * float32(s)
	zx := plane[1] * float32(s)
	xy := plane[2] * float32(s)
	return Rot3{yz, zx, xy, float32(c)}
}

// TODO: rename this so that it's clear that it's just purely numerical thing
// and doesn't actually change the rotation that's supposed to be represented.
func (a Rot3) Normalize() Rot3 {
	return Rot3(Vec4(a).NormalizeOr(Vec4(Rot3One())))
}

func (a Rot3) Inverse() Rot3 {
	return Rot3(quat[float32](a).Conj())
}

func (a Rot3) Mul(b Rot3) (ab Rot3) {
	// TODO: normalize as well?
	return Rot3(quat[float32](a).Mul(quat[float32](b)))
}

// TODO: kill this method in favor of TRS to affine conversion rolling its own
// specialization
func (r Rot3) ToMat() Mat3x3 {
	// ughhhhhhhhhhhhhhhhhhhhhh
	var R Mat3x3
	*R.Index(0, 0) = r[3]*r[3] + r[0]*r[0] - r[1]*r[1] - r[2]*r[2]
	*R.Index(0, 1) = r[0]*r[1]*2 - r[3]*r[2]*2
	*R.Index(0, 2) = r[3]*r[1]*2 + r[0]*r[2]*2
	*R.Index(1, 0) = r[3]*r[2]*2 + r[0]*r[1]*2
	*R.Index(1, 1) = r[3]*r[3] - r[0]*r[0] + r[1]*r[1] - r[2]*r[2]
	*R.Index(1, 2) = r[1]*r[2]*2 - r[3]*r[0]*2
	*R.Index(2, 0) = r[0]*r[2]*2 - r[3]*r[1]*2
	*R.Index(2, 1) = r[3]*r[0]*2 + r[1]*r[2]*2
	*R.Index(2, 2) = r[3]*r[3] - r[0]*r[0] - r[1]*r[1] + r[2]*r[2]
	return R
}

// TODO: introduce SLerp which will transparently use NLerp when estimated error
// is below some threshold?
func (a Rot3) NLerp(b Rot3, t float32) Rot3 {
	u, v := 1-t, t
	if Vec4(a).Dot(Vec4(b)) < 0 {
		v = -t
	}

	var c Rot3
	for i := range 4 {
		c[i] = a[i]*u + b[i]*v
	}
	return c.Normalize()
}

// TODO: rename to Rotate32 probably
func (a Rot3) Rotate(v Vec3) Vec3 {
	q := quat[float32](a)
	return q.Mul(quatFromVec3(v)).Mul(q.Conj()).Imag()
}

func (a Rot3) Rotate64(v DVec3) DVec3 {
	q := quat[float64](convert4[float64](a))
	return q.Mul(quatFromVec3(v)).Mul(q.Conj()).Imag()
}
