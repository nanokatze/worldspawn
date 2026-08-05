package gmath

import "golang.org/x/exp/constraints"

//go:generate go run ./_gen -o gmath_generated.go

// TODO: remove "To" from all conversion functions
// TODO: rename TRSD[T].Compose to TRSD[T].Affine

func triangularNumber(n int) int {
	return n * (n + 1) / 2
}

func Affine3FromMat[T constraints.Float](M Mat4x4[T]) Affine3[T] {
	var f Affine3[T]
	for i := range 3 {
		for j := range 3 {
			*f.M.Index(i, j) = float32(*M.Index(i, j))
		}
		f.T[i] = *M.Index(i, 3)
	}
	return f
}

func (f Affine3[T]) Mat() Mat4x4[T] {
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
	return Affine3FromMat(f.Mat().Inv())
}
