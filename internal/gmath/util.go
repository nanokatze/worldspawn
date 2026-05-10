package gmath

import (
	"math"

	"golang.org/x/exp/constraints"
)

// TODO: kill this and leave it up to the user
func Radians(x float64) float64 {
	const k float64 = math.Pi / 180
	return x * k
}

// TODO: add xy < 0 handling from https://www.open-std.org/jtc1/sc22/wg21/docs/papers/2019/p0811r3.html ?
func lerp[T constraints.Float](x, y T, t T) T {
	if t == 1 {
		return y
	}
	return x + (y-x)*t
}
