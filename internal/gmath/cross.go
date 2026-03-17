package gmath

func (a gvec3[T]) Cross(b gvec3[T]) (ab gvec3[T]) {
	return gvec3[T]{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
}
