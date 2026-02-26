package gmath

func (a vec3[T]) Cross(b vec3[T]) (ab vec3[T]) {
	return vec3[T]{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
}
