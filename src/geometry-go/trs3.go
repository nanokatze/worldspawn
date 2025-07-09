package geometry

import "structs"

type DTRS3 struct {
	_           structs.HostLayout
	Translation DVec3
	Rotation    Rot3
	Scale       Vec3
}

type TRS3 struct {
	_           structs.HostLayout
	Translation Vec3
	Rotation    Rot3
	Scale       Vec3
}

func (A TRS3) Inverse() TRS3 {
	panic("not implemented")
}

// Composition is not implemented as it can lead to shearing, which TRS3 can't
// represent.

func (A TRS3) NLerp(B TRS3, t float32) TRS3 {
	return TRS3{
		Translation: A.Translation.Lerp(B.Translation, t),
		Rotation:    A.Rotation.NLerp(B.Rotation, t),
		Scale:       A.Scale.Lerp(B.Scale, t),
	}
}

func (trs TRS3) Mat4x4() Mat4x4 {
	t := trs.Translation
	r := trs.Rotation
	s := trs.Scale

	// TODO: see if we can rewrite this to be more data driven? Especially the quaternion bit.
	// TODO: let's just have a Mat4x4() per each component and then compose them
	// together?

	var A Mat4x4

	A[0][0] = s[0] * (r[3]*r[3] + r[0]*r[0] - r[1]*r[1] - r[2]*r[2])
	A[0][1] = s[1] * (r[0]*r[1]*2 - r[3]*r[2]*2)
	A[0][2] = s[2] * (r[3]*r[1]*2 + r[0]*r[2]*2)

	A[1][0] = s[0] * (r[3]*r[2]*2 + r[0]*r[1]*2)
	A[1][1] = s[1] * (r[3]*r[3] - r[0]*r[0] + r[1]*r[1] - r[2]*r[2])
	A[1][2] = s[2] * (r[1]*r[2]*2 - r[3]*r[0]*2)

	A[2][0] = s[0] * (r[0]*r[2]*2 - r[3]*r[1]*2)
	A[2][1] = s[1] * (r[3]*r[0]*2 + r[1]*r[2]*2)
	A[2][2] = s[2] * (r[3]*r[3] - r[0]*r[0] - r[1]*r[1] + r[2]*r[2])

	for i := 0; i < 3; i++ {
		A[i][3] = t[i]
	}

	A[3][3] = 1

	return A
}
