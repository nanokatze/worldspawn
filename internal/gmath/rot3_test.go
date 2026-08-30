package gmath

import (
	"fmt"
	"slices"
	"testing"
)

func TestRot3AToB(t *testing.T) {
	const errorThreshold = 1e-6

	vectors := slices.Collect(
		func(yield func(Vec3f32) bool) {
			for i := range 3 {
				for s := range 2 {
					var v Vec3f32
					v[i] = 2*float32(s) - 1
					yield(v)
				}
			}
		})

	for i, a := range vectors {
		for j, b := range vectors {
			if a.Dot(b) == -1 {
				continue
			}

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
	a := Vec3f32{1, 0, 0}
	b := Vec3f32{0, 1, 0}
	R_base := Rot3AToB(a, b)

	rot3Pown := func(R Rot3, n int) Rot3 {
		A := Rot3One()
		for range n {
			A = A.Mul(R)
		}
		return A
	}

	t.Run("integer powers", func(t *testing.T) {
		for p := range 10 {
			R := rot3Pown(R_base, p)
			S := R_base.Pow(float32(p))

			t.Logf("R_%d = %v", p, R)
			t.Logf("S_%d = %v", p, S)
			diff := R.Mul(S.Inv())
			if *diff.CosAngle() != 1 {
				t.Error("significant difference")
			}
		}
	})
}
