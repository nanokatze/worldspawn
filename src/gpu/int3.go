package gpu

// TODO: move to util?
type int3 [3]int

func (a int3) Add(b int3) int3 {
	return int3{a[0] + b[0], a[1] + b[1], a[2] + b[2]}
}

func (a int3) Sub(b int3) int3 {
	return int3{a[0] - b[0], a[1] - b[1], a[2] - b[2]}
}

func (a int3) Mul(b int3) int3 {
	return int3{a[0] * b[0], a[1] * b[1], a[2] * b[2]}
}

func (a int3) Div(b int3) int3 {
	return int3{a[0] / b[0], a[1] / b[1], a[2] / b[2]}
}

func (a int3) Mod(b int3) int3 {
	return int3{a[0] % b[0], a[1] % b[1], a[2] % b[2]}
}

func (a int3) Rsh(b int3) int3 {
	return int3{a[0] >> b[0], a[1] >> b[1], a[2] >> b[2]}
}

func int3Min(a, b int3) int3 {
	return int3{min(a[0], b[0]), min(a[1], b[1]), min(a[2], b[2])}
}

func int3Max(a, b int3) int3 {
	return int3{max(a[0], b[0]), max(a[1], b[1]), max(a[2], b[2])}
}
