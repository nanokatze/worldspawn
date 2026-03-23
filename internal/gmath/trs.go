package gmath

import "golang.org/x/exp/constraints"

// Transitional stuff, TODO: remove

func GAffine3FromMat4x4[T constraints.Float](M gmat4x4[T]) GAffine3[T] {
	var f GAffine3[T]
	for i := range 3 {
		for j := range 3 {
			*f.M.Index(i, j) = float32(*M.Index(i, j))
		}
		f.T[i] = *M.Index(i, 3)
	}
	return f
}

func (f GAffine3[T]) Mat() gmat4x4[T] {
	var M gmat4x4[T]
	for i := range 3 {
		for j := range 3 {
			*M.Index(i, j) = T(*f.M.Index(i, j))
		}
		*M.Index(i, 3) = f.T[i]
	}
	*M.Index(3, 3) = 1
	return M
}

// TODO: _gen/affine.go should generate this
func (f GAffine3[T]) Inv() GAffine3[T] {
	return GAffine3FromMat4x4(f.Mat().Inverse())
}

// TODO: remove TRS types in favor of Affine once we implement
// factoring/decomposition and thus can comfortably implement interpolation

type DTRS3 struct {
	T DVec3
	R Rot3
	S Vec3
}

func DTRS3One() DTRS3 {
	return DTRS3{
		T: DVec3{},
		R: Rot3One(),
		S: Vec3Ones(),
	}
}

// TODO: pure rotation and pure scale DTRS3 constructors

func (x DTRS3) Mul(y DTRS3) DTRS3 {
	return DTRS3{
		T: x.T.Add(x.R.Rotate64(y.T)),
		R: x.R.Mul(y.R),
		S: Vec3Ones(),
	}
}

type TRS3 struct {
	T Vec3
	R Rot3
	S Vec3
}

func (A TRS3) Inverse() TRS3 {
	panic("not implemented")
}

func (A TRS3) NLerp(B TRS3, t float32) TRS3 {
	return TRS3{
		T: A.T.Lerp(B.T, t),
		R: A.R.NLerp(B.R, t),
		S: A.S.Lerp(B.S, t),
	}
}

func (trs TRS3) Mat4x4() Mat4x4 {
	t := trs.T
	r := trs.R
	s := trs.S

	// TODO: see if we can rewrite this to be more data driven? Especially the rotation bit.
	// TODO: let's just have a Mat4x4() per each component and then compose them
	// together?

	var A Mat4x4

	*A.Index(0, 0) = s[0] * (r[3]*r[3] + r[0]*r[0] - r[1]*r[1] - r[2]*r[2])
	*A.Index(0, 1) = s[1] * (r[0]*r[1]*2 - r[3]*r[2]*2)
	*A.Index(0, 2) = s[2] * (r[3]*r[1]*2 + r[0]*r[2]*2)

	*A.Index(1, 0) = s[0] * (r[3]*r[2]*2 + r[0]*r[1]*2)
	*A.Index(1, 1) = s[1] * (r[3]*r[3] - r[0]*r[0] + r[1]*r[1] - r[2]*r[2])
	*A.Index(1, 2) = s[2] * (r[1]*r[2]*2 - r[3]*r[0]*2)

	*A.Index(2, 0) = s[0] * (r[0]*r[2]*2 - r[3]*r[1]*2)
	*A.Index(2, 1) = s[1] * (r[3]*r[0]*2 + r[1]*r[2]*2)
	*A.Index(2, 2) = s[2] * (r[3]*r[3] - r[0]*r[0] - r[1]*r[1] + r[2]*r[2])

	for i := 0; i < 3; i++ {
		*A.Index(i, 3) = t[i]
	}

	*A.Index(3, 3) = 1

	return A
}
