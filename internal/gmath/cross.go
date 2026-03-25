package gmath

func (a Vec3[T]) Cross(b Vec3[T]) (ab Vec3[T]) {
	return Vec3[T]{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
}
