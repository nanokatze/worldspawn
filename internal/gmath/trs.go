package gmath

// TODO: kill this file!!!!!!!!!!

type DTRS3 struct {
	T DVec3
	R Rot3
	S Vec3
}

func DTRS3One() DTRS3 {
	return DTRS3{
		T: DVec3{},
		R: Rot3One(),
		S: Vec3Ones[float32](),
	}
}

// TODO: pure rotation and pure scale DTRS3 constructors

func (x DTRS3) Mul(y DTRS3) DTRS3 {
	return DTRS3{
		T: x.T.Add(x.R.Rotate64(y.T)),
		R: x.R.Mul(y.R),
		S: Vec3Ones[float32](),
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

func (trs TRS3) ToMat() Mat4x4 {
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
