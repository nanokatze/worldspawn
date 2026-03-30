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

// TODO: make this public?
// TODO: make sure the numeric properties of this are nice
func lerp[T constraints.Float](x, y T, t T) T {
	return x*(T(1)-t) + y*t
}
