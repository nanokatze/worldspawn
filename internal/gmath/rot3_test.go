package gmath

import (
	"fmt"
	"math"
	"slices"
	"testing"
)

func TestRot3AToB(t *testing.T) {
	const errorThreshold = 1e-6

	vectors := slices.Collect(
		func(yield func(Vec3f32) bool) {
			for i := range 3 {
				for s := range 1 {
					var v Vec3f32
					v[i] = float32(math.Pow(-1, float64(s)))
					yield(v)
				}
			}
		})

	for i, a := range vectors {
		for j, b := range vectors {
			t.Run(fmt.Sprintf("%d %d", i, j), func(t *testing.T) {
				t.Logf("a=%v", a)
				t.Logf("b=%v", b)
				R := Rot3AToB(a, b)
				t.Logf("R=%v", R)
				R_Rotate_a := R.Rotate(a)
				err := R_Rotate_a.Sub(b).Length()
				t.Logf("R.Rotate(a) = %v, want %v, err=%v", R_Rotate_a, b, err)
				if !(err < errorThreshold) {
					t.Error("error too large")
				}
			})
		}
	}
}

func TestRot3Pow(t *testing.T) {
}
