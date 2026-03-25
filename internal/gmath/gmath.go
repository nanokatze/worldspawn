package gmath

import "golang.org/x/exp/constraints"

//go:generate go run ./_gen -o gmath_generated.go

func triangularNumber(n int) int {
	return n * (n + 1) / 2
}

// TODO: kill this method in favor of TRS to affine conversion rolling its own
// specialization
func (S Shcale3) ToMat() Mat3x3f32 {
	return Mat3x3Uf32(S).ToMat()
}

// TODO: leave it up to the user to implement lerp?
func (A Shcale3) Lerp(B Shcale3, t float32) Shcale3 {
	var C Shcale3
	for i := range C {
		C[i] = lerp(A[i], B[i], t)
	}
	return C
}

// Transitional stuff, TODO: remove

func GAffine3FromMat4x4[T constraints.Float](M Mat4x4[T]) Affine3[T] {
	var f Affine3[T]
	for i := range 3 {
		for j := range 3 {
			*f.M.Index(i, j) = float32(*M.Index(i, j))
		}
		f.T[i] = *M.Index(i, 3)
	}
	return f
}

func (f Affine3[T]) ToMat() Mat4x4[T] {
	var M Mat4x4[T]
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
func (f Affine3[T]) Inv() Affine3[T] {
	return GAffine3FromMat4x4(f.ToMat().Inverse())
}

// TODO: delegate interpolation to the user?
func (A TRHS3[T]) NLerp(B TRHS3[T], t float32) TRHS3[T] {
	return TRHS3[T]{
		T: A.T.Lerp(B.T, T(t)),
		R: A.R.NLerp(B.R, t),
		S: A.S.Lerp(B.S, t),
	}
}
