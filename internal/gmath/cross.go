package gmath

// TODO: game and renderer should probably use their own and separate math packages?

//go:generate go run ./_gen -o geometry.go

func (a vec3[T]) Cross(b vec3[T]) (ab vec3[T]) {
	return vec3[T](a.Wedge(b))
}
