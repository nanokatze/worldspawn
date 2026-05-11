package gmath

import (
	"log"
	"testing"
)

func TestXxx(t *testing.T) {
	a := Vec3f32{1, 0, 0}
	b := Vec3f32{0, 1, 0}.Normalize()
	r := Rot3AToB(a, b)

	log.Println(r, r.Sqrt())

	epsilon := float32(1e-7)
	_ = epsilon

	for _, p := range []float32{0, 1, 2} {
		rp := r.Pow(p)
		t.Log(rp, rp.Rotate(a))
	}
}
