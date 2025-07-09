package geometry

import "math"

type Mat4x4 [4][4]float32

func Mat4x4Identity() (A Mat4x4) {
	for i := 0; i < 4; i++ {
		A[i][i] = 1
	}
	return A
}

// TODO: make it clear that it's reverse-z
// TODO: make the api be closer to VkViewport, so we could specify a "slice" of
// frustum (not centered around 0) ?
// TODO: we'll need to review our math with RT
func Mat4x4InfinitePerspective(fov, aspect, near float32) Mat4x4 {
	return Mat4x4{
		{1 / float32(math.Tan(float64(fov/2))) / aspect, 0, 0, 0},
		{0, 1 / float32(math.Tan(float64(fov/2))), 0, 0},
		{0, 0, 0, near},
		{0, 0, -1, 0},
	}
}

func (A Mat4x4) Mul4x4(B Mat4x4) (C Mat4x4) {
	// TODO: replace this with a more general gemm/matmul
	// NOTE: Go doesn't unroll loops for now and in the future will unroll only
	// with PGO, see what we can do about that.
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				C[i][j] += A[i][k] * B[k][j]
			}
		}
	}
	return C
}

func (A Mat4x4) Inverse() (A_inv Mat4x4) {
	// TODO: write a generalized routine for computing inverse that can be
	// unrolled and use it to replace this thing

	A2323 := A[2][2]*A[3][3] - A[2][3]*A[3][2]
	A1323 := A[2][1]*A[3][3] - A[2][3]*A[3][1]
	A1223 := A[2][1]*A[3][2] - A[2][2]*A[3][1]
	A0323 := A[2][0]*A[3][3] - A[2][3]*A[3][0]
	A0223 := A[2][0]*A[3][2] - A[2][2]*A[3][0]
	A0123 := A[2][0]*A[3][1] - A[2][1]*A[3][0]
	A2313 := A[1][2]*A[3][3] - A[1][3]*A[3][2]
	A1313 := A[1][1]*A[3][3] - A[1][3]*A[3][1]
	A1213 := A[1][1]*A[3][2] - A[1][2]*A[3][1]
	A2312 := A[1][2]*A[2][3] - A[1][3]*A[2][2]
	A1312 := A[1][1]*A[2][3] - A[1][3]*A[2][1]
	A1212 := A[1][1]*A[2][2] - A[1][2]*A[2][1]
	A0313 := A[1][0]*A[3][3] - A[1][3]*A[3][0]
	A0213 := A[1][0]*A[3][2] - A[1][2]*A[3][0]
	A0312 := A[1][0]*A[2][3] - A[1][3]*A[2][0]
	A0212 := A[1][0]*A[2][2] - A[1][2]*A[2][0]
	A0113 := A[1][0]*A[3][1] - A[1][1]*A[3][0]
	A0112 := A[1][0]*A[2][1] - A[1][1]*A[2][0]

	invDet := A[0][0]*(A[1][1]*A2323-A[1][2]*A1323+A[1][3]*A1223) -
		A[0][1]*(A[1][0]*A2323-A[1][2]*A0323+A[1][3]*A0223) +
		A[0][2]*(A[1][0]*A1323-A[1][1]*A0323+A[1][3]*A0123) -
		A[0][3]*(A[1][0]*A1223-A[1][1]*A0223+A[1][2]*A0123)
	det := 1 / invDet

	return Mat4x4{
		{
			det * (A[1][1]*A2323 - A[1][2]*A1323 + A[1][3]*A1223),
			det * -(A[0][1]*A2323 - A[0][2]*A1323 + A[0][3]*A1223),
			det * (A[0][1]*A2313 - A[0][2]*A1313 + A[0][3]*A1213),
			det * -(A[0][1]*A2312 - A[0][2]*A1312 + A[0][3]*A1212),
		},
		{
			det * -(A[1][0]*A2323 - A[1][2]*A0323 + A[1][3]*A0223),
			det * (A[0][0]*A2323 - A[0][2]*A0323 + A[0][3]*A0223),
			det * -(A[0][0]*A2313 - A[0][2]*A0313 + A[0][3]*A0213),
			det * (A[0][0]*A2312 - A[0][2]*A0312 + A[0][3]*A0212),
		},
		{
			det * (A[1][0]*A1323 - A[1][1]*A0323 + A[1][3]*A0123),
			det * -(A[0][0]*A1323 - A[0][1]*A0323 + A[0][3]*A0123),
			det * (A[0][0]*A1313 - A[0][1]*A0313 + A[0][3]*A0113),
			det * -(A[0][0]*A1312 - A[0][1]*A0312 + A[0][3]*A0112),
		},
		{
			det * -(A[1][0]*A1223 - A[1][1]*A0223 + A[1][2]*A0123),
			det * (A[0][0]*A1223 - A[0][1]*A0223 + A[0][2]*A0123),
			det * -(A[0][0]*A1213 - A[0][1]*A0213 + A[0][2]*A0113),
			det * (A[0][0]*A1212 - A[0][1]*A0212 + A[0][2]*A0112),
		},
	}
}

// TODO: internal api to view matrices and vectors as []float32 or std::mdspan

/*
func matmul(A, B, C []float32) {

}
*/
