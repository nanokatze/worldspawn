package gmath

import "math"

// TODO: make it clear that it's reverse-z
// TODO: make the api be closer to VkViewport, so we could specify a "slice" of
// frustum (not centered around 0) ?
// TODO: we'll need to review our math with RT
// TODO: move this out of here and into pathtracer?
func Mat4x4InfinitePerspective(fov, aspect, near float32) Mat4x4f32 {
	return Mat4x4f32{
		1 / float32(math.Tan(float64(fov/2))) / aspect, 0, 0, 0,
		0, 1 / float32(math.Tan(float64(fov/2))), 0, 0,
		0, 0, 0, near,
		0, 0, -1, 0,
	}
}

// TODO: generate this
func (A Mat4x4[T]) Inverse() (A_inv Mat4x4[T]) {
	// TODO: write a generalized routine for computing inverse that can be
	// unrolled and use it to replace this thing

	A2323 := *A.Index(2, 2)**A.Index(3, 3) - *A.Index(2, 3)**A.Index(3, 2)
	A1323 := *A.Index(2, 1)**A.Index(3, 3) - *A.Index(2, 3)**A.Index(3, 1)
	A1223 := *A.Index(2, 1)**A.Index(3, 2) - *A.Index(2, 2)**A.Index(3, 1)
	A0323 := *A.Index(2, 0)**A.Index(3, 3) - *A.Index(2, 3)**A.Index(3, 0)
	A0223 := *A.Index(2, 0)**A.Index(3, 2) - *A.Index(2, 2)**A.Index(3, 0)
	A0123 := *A.Index(2, 0)**A.Index(3, 1) - *A.Index(2, 1)**A.Index(3, 0)
	A2313 := *A.Index(1, 2)**A.Index(3, 3) - *A.Index(1, 3)**A.Index(3, 2)
	A1313 := *A.Index(1, 1)**A.Index(3, 3) - *A.Index(1, 3)**A.Index(3, 1)
	A1213 := *A.Index(1, 1)**A.Index(3, 2) - *A.Index(1, 2)**A.Index(3, 1)
	A2312 := *A.Index(1, 2)**A.Index(2, 3) - *A.Index(1, 3)**A.Index(2, 2)
	A1312 := *A.Index(1, 1)**A.Index(2, 3) - *A.Index(1, 3)**A.Index(2, 1)
	A1212 := *A.Index(1, 1)**A.Index(2, 2) - *A.Index(1, 2)**A.Index(2, 1)
	A0313 := *A.Index(1, 0)**A.Index(3, 3) - *A.Index(1, 3)**A.Index(3, 0)
	A0213 := *A.Index(1, 0)**A.Index(3, 2) - *A.Index(1, 2)**A.Index(3, 0)
	A0312 := *A.Index(1, 0)**A.Index(2, 3) - *A.Index(1, 3)**A.Index(2, 0)
	A0212 := *A.Index(1, 0)**A.Index(2, 2) - *A.Index(1, 2)**A.Index(2, 0)
	A0113 := *A.Index(1, 0)**A.Index(3, 1) - *A.Index(1, 1)**A.Index(3, 0)
	A0112 := *A.Index(1, 0)**A.Index(2, 1) - *A.Index(1, 1)**A.Index(2, 0)

	invDet := *A.Index(0, 0)*(*A.Index(1, 1)*A2323-*A.Index(1, 2)*A1323+*A.Index(1, 3)*A1223) -
		*A.Index(0, 1)*(*A.Index(1, 0)*A2323-*A.Index(1, 2)*A0323+*A.Index(1, 3)*A0223) +
		*A.Index(0, 2)*(*A.Index(1, 0)*A1323-*A.Index(1, 1)*A0323+*A.Index(1, 3)*A0123) -
		*A.Index(0, 3)*(*A.Index(1, 0)*A1223-*A.Index(1, 1)*A0223+*A.Index(1, 2)*A0123)
	det := 1 / invDet

	return Mat4x4[T]{
		det * (*A.Index(1, 1)*A2323 - *A.Index(1, 2)*A1323 + *A.Index(1, 3)*A1223),
		det * -(*A.Index(0, 1)*A2323 - *A.Index(0, 2)*A1323 + *A.Index(0, 3)*A1223),
		det * (*A.Index(0, 1)*A2313 - *A.Index(0, 2)*A1313 + *A.Index(0, 3)*A1213),
		det * -(*A.Index(0, 1)*A2312 - *A.Index(0, 2)*A1312 + *A.Index(0, 3)*A1212),
		det * -(*A.Index(1, 0)*A2323 - *A.Index(1, 2)*A0323 + *A.Index(1, 3)*A0223),
		det * (*A.Index(0, 0)*A2323 - *A.Index(0, 2)*A0323 + *A.Index(0, 3)*A0223),
		det * -(*A.Index(0, 0)*A2313 - *A.Index(0, 2)*A0313 + *A.Index(0, 3)*A0213),
		det * (*A.Index(0, 0)*A2312 - *A.Index(0, 2)*A0312 + *A.Index(0, 3)*A0212),
		det * (*A.Index(1, 0)*A1323 - *A.Index(1, 1)*A0323 + *A.Index(1, 3)*A0123),
		det * -(*A.Index(0, 0)*A1323 - *A.Index(0, 1)*A0323 + *A.Index(0, 3)*A0123),
		det * (*A.Index(0, 0)*A1313 - *A.Index(0, 1)*A0313 + *A.Index(0, 3)*A0113),
		det * -(*A.Index(0, 0)*A1312 - *A.Index(0, 1)*A0312 + *A.Index(0, 3)*A0112),
		det * -(*A.Index(1, 0)*A1223 - *A.Index(1, 1)*A0223 + *A.Index(1, 2)*A0123),
		det * (*A.Index(0, 0)*A1223 - *A.Index(0, 1)*A0223 + *A.Index(0, 2)*A0123),
		det * -(*A.Index(0, 0)*A1213 - *A.Index(0, 1)*A0213 + *A.Index(0, 2)*A0113),
		det * (*A.Index(0, 0)*A1212 - *A.Index(0, 1)*A0212 + *A.Index(0, 2)*A0112),
	}
}
