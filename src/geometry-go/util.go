package geometry

import "math"

func Radians(x float64) float64 {
	const k float64 = math.Pi / 180
	return x * k
}

// TODO: make this public?
func lerp32(a, b, t float32) float32 {
	// TODO: should be expressed in terms of FMA for nice rounding.
	return a + (b-a)*t
	// return math.FMA(b, float64(t), math.FMA())
}
