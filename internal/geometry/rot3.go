package geometry

import (
	"math"
)

// TODO: kill quat and generate RotN types

type Rot3 [4]float32

func Rot3One() Rot3 {
	return Rot3{0, 0, 0, 1}
}

func Rot3InPlane(plane Bivec3, θ float32) Rot3 {
	s, c := math.Sincos(float64(θ / 2))
	yz := plane[0] * float32(s)
	zx := plane[1] * float32(s)
	xy := plane[2] * float32(s)
	return Rot3{yz, zx, xy, float32(c)}
}

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

// TODO: conversion routine to Mat4x4
