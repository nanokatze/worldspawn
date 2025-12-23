package gpu

type int3 [3]int

func (x int3) Add(y int3) int3 {
	return int3{x[0] + y[0], x[1] + y[1], x[2] + y[2]}
}

func (x int3) Sub(y int3) int3 {
	return int3{x[0] - y[0], x[1] - y[1], x[2] - y[2]}
}

func (x int3) Mul(y int3) int3 {
	return int3{x[0] * y[0], x[1] * y[1], x[2] * y[2]}
}

func (x int3) Div(y int3) int3 {
	return int3{x[0] / y[0], x[1] / y[1], x[2] / y[2]}
}

func (x int3) Mod(y int3) int3 {
	return int3{x[0] % y[0], x[1] % y[1], x[2] % y[2]}
}

func (x int3) Rsh(y int3) int3 {
	return int3{x[0] >> y[0], x[1] >> y[1], x[2] >> y[2]}
}

func (x int3) Min(y int3) int3 {
	return int3{min(x[0], y[0]), min(x[1], y[1]), min(x[2], y[2])}
}

func (x int3) Max(y int3) int3 {
	return int3{max(x[0], y[0]), max(x[1], y[1]), max(x[2], y[2])}
}
