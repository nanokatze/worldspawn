package apostprocess

import "worldspawn/internal/apostprocess/interpolators"

// TODO: ok I guess we'll just use the 3d sound renderer or whatever for UI
// sounds and OST as well.
//
// We'll still need a mixer with tracks and clips and effects for adaptive
// sounds and stuff. I guess clips should be opaque and contain data for
// different instruments (e.g. playing back a file would be an instrument) with
// some arguments.

func Resample(dst []float32, src []float32, channels int, ratio float64) {
	for i := range len(dst) / channels {
		for channel := range channels {
			t := float64(i) * ratio
			j := int(t)
			dst[i*channels+channel] = interpolators.LagrangeP4O3(
				sliceLoadOrZero(src, (j-1)*channels+channel),
				sliceLoadOrZero(src, (j+0)*channels+channel),
				sliceLoadOrZero(src, (j+1)*channels+channel),
				sliceLoadOrZero(src, (j+2)*channels+channel),
				float32(t-float64(j)))
		}
	}
}

func sliceLoadOrZero[T any](x []T, i int) T {
	if !(0 <= i && i < len(x)) {
		return *new(T)
	}
	return x[i]
}
