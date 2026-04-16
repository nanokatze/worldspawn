package arenderer

import (
	"worldspawn/internal/gmath"
)

// TODO: duplicate Geometry from grenderer

type Scene struct {
	Transform []gmath.TRS3f32

	// geometry []grenderer.Geometry

	Emitters [][]float32 // []func(buf []float32)
}

func NewScene(n int) *Scene {
	return &Scene{
		Transform: make([]gmath.TRS3f32, n),

		Emitters: make([][]float32, n),
	}
}

var earDirs = [2]gmath.Vec3f32{
	{-1, 0, 0},
	{1, 0, 0},
}

// TODO: it would probably make more sense to get cameraTransform decomposed...
// But I guess whatever
func (scene *Scene) Render(film Film, cameraTransform gmath.Affine3f32) {
	ugh := cameraTransform.TRS().R.Inverse()

	for i, emitter := range scene.Emitters {
		if len(emitter) == 0 {
			continue
		}

		AB := ugh.Rotate(scene.Transform[i].T.Sub(cameraTransform.T))

		d := AB.Length()
		// TODO: account for stokes' extinction law as well
		// TODO: this is not a good approximation of % of energy we're getting
		x := 1.0 / max(d*d, 1)

		for k := range film.Channels {
			x2 := min(1+earDirs[k].Dot(AB.Normalize()), 1) * x

			for j := range len(film.Samples) / film.Channels {
				film.Samples[j*film.Channels+k] += x2 * sliceLoadOrZero(emitter, j)
			}
		}
	}
}

func sliceLoadOrZero[T any](s []T, i int) T {
	if !(0 <= i && i < len(s)) {
		return *new(T)
	}
	return s[i]
}
