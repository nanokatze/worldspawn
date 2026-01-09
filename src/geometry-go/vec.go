package geometry

// TODO: game and renderer should probably use their own and separate math packages?

//go:generate go run gen_vec.go -o gvec2.go -D 2

type Vec2 = gvec2[float32]

func Vec2Ones() Vec2 { return gvec2Ones[float32]() }

//go:generate go run gen_vec.go -o gvec3.go -D 3

func (a gvec3[T]) Cross(b gvec3[T]) (ab gvec3[T]) {
	return gvec3[T]{
		a[1]*b[2] + a[2]*b[1],
		a[2]*b[0] + a[0]*b[2],
		a[0]*b[1] + a[1]*b[0],
	}
}

type Vec3 = gvec3[float32]

func Vec3Ones() Vec3 { return gvec3Ones[float32]() }

type DVec3 = gvec3[float64]

func DVec3Ones() DVec3 { return gvec3Ones[float64]() }

//go:generate go run gen_vec.go -o gvec4.go -D 4

type Vec4 = gvec4[float32]

func Vec4Ones() Vec4 { return gvec4Ones[float32]() }
