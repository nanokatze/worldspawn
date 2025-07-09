package geometry

import (
	"math"
)

// TODO: rename this package to linalg?

type Rot3 [4]float32

// TODO: rename to Rot3One?
func Rot3One() Rot3 {
	return Rot3{0, 0, 0, 1}
}

func Rot3FromPlaneAngle(b Vec3, θ float32) Rot3 {
	sinHalfθ, cosHalfθ := math.Sincos(float64(θ / 2))
	yz := b[0] * float32(sinHalfθ)
	zx := b[1] * float32(sinHalfθ)
	xy := b[2] * float32(sinHalfθ)
	return Rot3{yz, zx, xy, float32(cosHalfθ)}
}

// TODO: should this be NormalizedOr too? No
func (a Rot3) Normalized() Rot3 {
	return Rot3(Vec4(a).NormalizedOr(Vec4(Rot3One())))
}

func (a Rot3) Inverse() Rot3 {
	return Rot3(quat(a).Conj())
}

func (a Rot3) Mul(b Rot3) (ab Rot3) {
	// TODO: normalize as well?
	return Rot3(quat(a).Mul(quat(b)))
}

// TODO: introduce SLerp which will transparently use NLerp when estimated error
// is below some threshold?
func (a Rot3) NLerp(b Rot3, t float32) Rot3 {
	// TODO: clean this up and make it more obvious what's happening here
	u, v := 1-t, t
	if Vec4(a).Dot(Vec4(b)) < 0 {
		v = -t
	}

	var c Rot3
	for i := 0; i < 4; i++ {
		c[i] = a[i]*u + b[i]*v
	}
	return c.Normalized()
}

// TODO: rename to Rotate32 probably
func (a Rot3) Rotate(v Vec3) Vec3 {
	q := quat(a)
	return q.Mul(quat{v[0], v[1], v[2], 0}).Mul(q.Conj()).Imag()
}

func (a Rot3) Rotate64(v DVec3) DVec3 {
	q := quat(a).DQuat()
	return q.Mul(dquat{v[0], v[1], v[2], 0}).Mul(q.Conj()).Imag()
}

// TODO: conversion routine to Mat4x4

var quatMulTab = [4][4]uint8{
	/*       i     j     k     1 */
	/*i*/ {0x83, 0x02, 0x81, 0x00},
	/*j*/ {0x82, 0x83, 0x00, 0x01},
	/*k*/ {0x01, 0x80, 0x83, 0x02},
	/*1*/ {0x00, 0x01, 0x02, 0x03},
}

type quat [4]float32

func (q quat) Conj() quat {
	return quat{-q[0], -q[1], -q[2], q[3]}
}

func (p quat) Mul(q quat) (pq quat) {
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			k := quatMulTab[i][j] & 3
			s := 1 - 2*float32(quatMulTab[i][j]>>7)
			pq[k] += s * (p[i] * q[j])
		}
	}
	return pq
}

func (q quat) Imag() Vec3 {
	return Vec3{q[0], q[1], q[2]}
}

func (q quat) DQuat() (q64 dquat) {
	for i := 0; i < 4; i++ {
		q64[i] = float64(q[i])
	}
	return q64
}

type dquat [4]float64

func (q dquat) Conj() dquat {
	return dquat{-q[0], -q[1], -q[2], q[3]}
}

func (p dquat) Mul(q dquat) (pq dquat) {
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			k := quatMulTab[i][j] & 3
			s := 1 - 2*float64(quatMulTab[i][j]>>7)
			pq[k] += s * (p[i] * q[j])
		}
	}
	return pq
}

func (q dquat) Imag() DVec3 {
	return DVec3{q[0], q[1], q[2]}
}
