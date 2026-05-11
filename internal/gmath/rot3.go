package gmath

import (
	"math"

	"golang.org/x/exp/constraints"
)

type Plane3[T constraints.Float] [3]T

type Plane3f32 = Plane3[float32]

func Plane3OnVectors[T constraints.Float](a, b Vec3[T]) Plane3[T] {
	return Plane3[T](a.Cross(b))
}

// TODO: kill quat and generate RotN types

type Rot3 [4]float32

func Rot3One() Rot3 {
	return Rot3{0, 0, 0, 1}
}

func Rot3InPlane(plane Plane3f32, θ float32) Rot3 {
	s, c := math.Sincos(float64(θ / 2))
	yz := plane[0] * float32(s)
	zx := plane[1] * float32(s)
	xy := plane[2] * float32(s)
	return Rot3{yz, zx, xy, float32(c)}
}

func Rot3FromMat(m Mat3x3f32) Rot3 {
	// TODO: generalized rewrite pls?

	r00 := *m.Index(0, 0)
	r01 := *m.Index(0, 1)
	r02 := *m.Index(0, 2)
	r10 := *m.Index(1, 0)
	r11 := *m.Index(1, 1)
	r12 := *m.Index(1, 2)
	r20 := *m.Index(2, 0)
	r21 := *m.Index(2, 1)
	r22 := *m.Index(2, 2)

	tw := 1.0 + r00 + r11 + r22
	tx := 1.0 + r00 - r11 - r22
	ty := 1.0 - r00 + r11 - r22
	tz := 1.0 - r00 - r11 + r22

	max_t := max(tw, tx, ty, tz)

	sqrt := func(x float32) float32 { return float32(math.Sqrt(float64(x))) }

	var w, x, y, z float32
	switch max_t {
	case tw:
		w = 0.5 * sqrt(tw)
		s := 0.25 / w
		x = (r21 - r12) * s
		y = (r02 - r20) * s
		z = (r10 - r01) * s
	case tx:
		x = 0.5 * sqrt(tx)
		s := 0.25 / x
		w = (r21 - r12) * s
		y = (r01 + r10) * s
		z = (r02 + r20) * s
	case ty:
		y = 0.5 * sqrt(ty)
		s := 0.25 / y
		w = (r02 - r20) * s
		x = (r01 + r10) * s
		z = (r12 + r21) * s
	case tz:
		z = 0.5 * sqrt(tz)
		s := 0.25 / z
		w = (r10 - r01) * s
		x = (r02 + r20) * s
		y = (r12 + r21) * s
	}

	r := Rot3{x, y, z, w}
	r.Renormalize()
	return r
}

func (a Rot3) Renormalize() Rot3 {
	tmp := Vec4f32(a)
	if !(tmp.Dot(tmp) > 0) {
		return Rot3One()
	}
	return Rot3(tmp.Normalize())
}

// TODO: rename to Inv()
func (a Rot3) Inverse() Rot3 {
	return Rot3(quat[float32](a).Conj())
}

func (a Rot3) Mul(b Rot3) (ab Rot3) {
	return Rot3(quat[float32](a).Mul(quat[float32](b))).Renormalize()
}

func (r Rot3) ToMat() Mat3x3f32 {
	// ughhhhhhhhhhhhhhhhhhhhhh
	var R Mat3x3f32
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
	if Vec4f32(a).Dot(Vec4f32(b)) < 0 {
		v = -t
	}

	var c Rot3
	for i := range 4 {
		c[i] = a[i]*u + b[i]*v
	}
	return c.Renormalize()
}

// TODO: rename to Rotate32 probably
func (a Rot3) Rotate(v Vec3f32) Vec3f32 {
	q := quat[float32](a)
	return q.Mul(quatFromVec3(v)).Mul(q.Conj()).Imag()
}

func (a Rot3) Rotate64(v Vec3f64) Vec3f64 {
	q := quat[float64](Vec4Convert[float64](Vec4f32(a)))
	return q.Mul(quatFromVec3(v)).Mul(q.Conj()).Imag()
}
