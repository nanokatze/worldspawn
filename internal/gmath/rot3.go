package gmath

import (
	"math"

	"golang.org/x/exp/constraints"
)

// TODO: kill quat and generate RotN types

// TODO: currently our rotations always double cover the SO(d). We should
// require that the cos(angle) is non-negative so that each set of coefficients
// corresponds to a unique rotation object and we can use == to compare
// rotations for equality. As an extension of this, we could also avoid storing
// the cos(angle) explicitly.

// TODO: offer constructors for Rot2 and Rot3 which convert from complex and
// quaternion respectively

type Rot3 [4]float32

func (R *Rot3) Fixed() *Vec3f32 { return (*Vec3f32)(R[1:4]) }

func (R *Rot3) CosAngle() *float32 { return &R[0] }

func Rot3One() Rot3 {
	var R Rot3
	*R.CosAngle() = 1
	return R
}

// Rot3AToB constructs a rotation R such that R.Rotate(a) == b. a and b must be
// unit.
// TODO: rename to Rot3AB?
func Rot3AToB(a, b Vec3f32) Rot3 {
	// TODO: handle the case when a.Dot(b) == -1?

	var RSqr Rot3
	*RSqr.CosAngle() = a.Dot(b)
	*RSqr.Fixed() = a.Cross(b)
	return RSqr.Sqrt()
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

	return Rot3{w, x, y, z}.Renormalize()
}

func (R Rot3) Renormalize() Rot3 {
	// TODO: quit relying on VecN
	tmp := Vec4f32(R)
	if !(tmp.Dot(tmp) > 0) {
		return Rot3One()
	}
	return Rot3(tmp.Normalize())
}

func (R Rot3) Mul(S Rot3) Rot3 {
	return Rot3(quat[float32](R).Mul(quat[float32](S))).Renormalize()
}

func (R Rot3) Inv() Rot3 {
	return Rot3(quat[float32](R).Conj())
}

func (R Rot3) Pow(p float32) Rot3 {
	// TODO: naming

	φ := float32(math.Acos(float64(*R.CosAngle())))
	B := R.Fixed().Normalize()

	pφ := p * φ

	sinpφ, cospφ := math.Sincos(float64(pφ))

	B2 := B.Scale(float32(sinpφ))

	return Rot3{float32(cospφ), B2[0], B2[1], B2[2]}
}

func (R Rot3) Sqrt() Rot3 {
	return Rot3{
		0.5*R[0] + float32(math.Copysign(0.5, float64(R[0]))),
		0.5 * R[1],
		0.5 * R[2],
		0.5 * R[3],
	}.Renormalize()
}

func (R Rot3) Rotate[T constraints.Float](v Vec3[T]) Vec3[T] {
	q := quat[T](Vec4f32(R).Convert[T]())
	return q.Mul(quatFromVec3(v)).Mul(q.Conj()).Imag()
}

func (R Rot3) Mat() Mat3x3f32 {
	var M Mat3x3f32
	for i := range 3 {
		v := Vec3f32{}
		v[i] = 1
		Rv := R.Rotate(v)
		for j := range 3 {
			*M.Index(j, i) = Rv[j]
		}
	}
	return M
}
