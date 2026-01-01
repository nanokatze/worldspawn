package geometry

import (
	"math"

	"golang.org/x/exp/constraints"
)

func Radians(x float64) float64 {
	const k float64 = math.Pi / 180
	return x * k
}

// TODO: make this public?
func lerp[T constraints.Float](x, y T, t T) T {
	return x*(T(1)-t) + y*t
}

func convert4[T, U constraints.Float](v [4]U) [4]T {
	return [4]T{T(v[0]), T(v[1]), T(v[2]), T(v[3])}
}
