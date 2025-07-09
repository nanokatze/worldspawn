package gpu

type Int3 struct {
	X, Y, Z int
}

// TODO: should these methods be public?
func (a Int3) Add(b Int3) Int3 {
	return Int3{a.X + b.X, a.Y + b.Y, a.Z + b.Z}
}

func (a Int3) Sub(b Int3) Int3 {
	return Int3{a.X - b.X, a.Y - b.Y, a.Z - b.Z}
}

func (a Int3) Mul(b Int3) Int3 {
	return Int3{a.X * b.X, a.Y * b.Y, a.Z * b.Z}
}

func (a Int3) Div(b Int3) Int3 {
	return Int3{a.X / b.X, a.Y / b.Y, a.Z / b.Z}
}

func (a Int3) Mod(b Int3) Int3 {
	return Int3{a.X % b.X, a.Y % b.Y, a.Z % b.Z}
}

func (a Int3) Rsh(b Int3) Int3 {
	return Int3{a.X >> b.X, a.Y >> b.Y, a.Z >> b.Z}
}

func int3Min(a, b Int3) Int3 {
	return Int3{min(a.X, b.X), min(a.Y, b.Y), min(a.Z, b.Z)}
}

func int3Max(a, b Int3) Int3 {
	return Int3{max(a.X, b.X), max(a.Y, b.Y), max(a.Z, b.Z)}
}
