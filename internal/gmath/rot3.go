package gmath

import (
	"math"
)

// Deprecated; TODO: kill this
type Plane3f32 [3]float32

func Plane3OnVectors(a, b Vec3f32) Plane3f32 {
	return Plane3f32(a.Cross(b))
}

// TODO: kill quat and generate RotN types

// TODO: move scalar to be at [0]
type Rot3 [4]float32

func Rot3One() Rot3 {
	return Rot3{0, 0, 0, 1}
}

// Rot3AToB constructs a rotation R such that R.Rotate(a) == b. a and b must be
// unit and (for now) a.Dot(b) must be non-negative.
func Rot3AToB(a, b Vec3f32) Rot3 {
	// TODO: handle the case when a.Dot(b) < 0

	// TODO: better naming for different grade elements

	B := a.Cross(b)
	scalar := float32(math.Sqrt(float64(1 - B.Dot(B))))
	return Rot3{B[0], B[1], B[2], scalar}.Sqrt()
}

// Deprecated; TODO: kill this
func Rot3InPlane(plane Plane3f32, θ float32) Rot3 {
	sinTheta, cosTheta := math.Sincos(float64(θ / 2))
	yz := plane[0] * float32(sinTheta)
	zx := plane[1] * float32(sinTheta)
	xy := plane[2] * float32(sinTheta)
	return Rot3{yz, zx, xy, float32(cosTheta)}
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

func (R Rot3) Renormalize() Rot3 {
	// TODO: quit relying on VecN
	tmp := Vec4f32(R)
	if !(tmp.Dot(tmp) > 0) {
		return Rot3One()
	}
	return Rot3(tmp.Normalize())
}

// TODO: rename to Inv()
func (R Rot3) Inverse() Rot3 {
	return Rot3(quat[float32](R).Conj())
}

func (R Rot3) Mul(R2 Rot3) Rot3 {
	return Rot3(quat[float32](R).Mul(quat[float32](R2))).Renormalize()
}

func (R Rot3) Pow(p float32) Rot3 {
	if p < 0 {
		R = R.Inverse()
		p = -p
	}

	φ := float32(math.Acos(float64(R[3])))
	B := Vec3f32(R[0:3]).Scale(1.0 / float32(math.Sin(float64(φ))))

	φ2 := p * φ

	sinPhi2, cosPhi2 := math.Sincos(float64(φ2))

	B2 := B.Scale(float32(sinPhi2))

	return Rot3{B2[0], B2[1], B2[2], float32(cosPhi2)}
}

func (R Rot3) Sqrt() Rot3 {
	// Gross; TODO: can we please just make composition and constructors maintain this?
	if R[3] < 0 {
		for i := range 4 {
			R[i] *= -1
		}
	}

	return Rot3{
		0.5 * R[0],
		0.5 * R[1],
		0.5 * R[2],
		0.5 + 0.5*R[3],
	}.Renormalize()
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
// TODO: kill this tbh, the user has Pow they can use to implement interpolation
func (R Rot3) NLerp(R2 Rot3, t float32) Rot3 {
	u, v := 1-t, t
	if Vec4f32(R).Dot(Vec4f32(R2)) < 0 {
		v = -t
	}

	var c Rot3
	for i := range 4 {
		c[i] = R[i]*u + R2[i]*v
	}
	return c.Renormalize()
}

// TODO: rename to Rotate32 probably, or actually make it generic!
func (R Rot3) Rotate(v Vec3f32) Vec3f32 {
	q := quat[float32](R)
	return q.Mul(quatFromVec3(v)).Mul(q.Conj()).Imag()
}

func (R Rot3) Rotate64(v Vec3f64) Vec3f64 {
	q := quat[float64](Vec4Convert[float64](Vec4f32(R)))
	return q.Mul(quatFromVec3(v)).Mul(q.Conj()).Imag()
}
