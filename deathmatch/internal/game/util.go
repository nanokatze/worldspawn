package game

import (
	"iter"
	"math"
	"time"

	"worldspawn/internal/gmath"

	"golang.org/x/exp/constraints"
)

func durationToFloatSeconds(d time.Duration) float64 {
	return float64(d/1e9) + float64(d%1e9)/1e9
}

// TODO: move these somewhere else

func sampleSphere(u gmath.Vec2f32) gmath.Vec3f32 {
	θ := float64(2 * math.Pi * u[0])
	φ := math.Acos(float64(1 - 2*u[1]))
	sinθ, cosθ := math.Sincos(θ)
	sinφ, cosφ := math.Sincos(φ)
	return gmath.Vec3f64{
		cosθ * sinφ,
		sinθ * sinφ,
		cosφ,
	}.Convert[float32]()
}

// TODO: https://extremelearning.com.au/evenly-distributing-points-on-a-sphere/
func fibonacciLattice(n int64) iter.Seq[gmath.Vec2f32] {
	goldenRatio := (1 + math.Sqrt(5)) / 2

	return func(yield func(gmath.Vec2f32) bool) {
		for i := range n {
			p := gmath.Vec2f64{math.Mod(float64(i)/goldenRatio, 1), float64(i) / float64(n-1)}
			if !yield(p.Convert[float32]()) {
				break
			}
		}
	}
}

func mix[T constraints.Float](a, b, c T) T {
	if c == 1 {
		return b
	}
	return a + (b-a)*c
}
