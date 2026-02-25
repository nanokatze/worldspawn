package main

import (
	"log"
	"math"
	"sync"
	"unsafe"

	sfx "worldspawn/internal/fuckwwise"
	"worldspawn/internal/fuckwwise/interpolators"
	"worldspawn/internal/gmath"
	"worldspawn/internal/sdl"
)

// TODO: output related stuff back to main.go?

const sampleRate = 48000

// TODO: naming
// TODO: make this non-fatal?
var au = sync.OnceValue(func() *sdl.AudioStream {
	au, err := sdl.OpenAudioDeviceStream(
		sdl.AUDIO_DEVICE_DEFAULT_PLAYBACK,
		&sdl.AudioSpec{
			Format:     sdl.AUDIO_F32,
			Channels:   2,
			SampleRate: sampleRate,
		})
	if err != nil {
		log.Fatal(err)
	}
	if err := au.Device().Resume(); err != nil {
		log.Fatal(err)
	}
	return au
})

var sources = make(map[string]*sfx.Source)

// TODO: make this possible to run async pls
func renderAudio(scene *sfx.Scene, camera gmath.Vec3, t0, Δt int64) {
	// TODO: make this tunable at runtime
	queueingTargetSamples := 48000 / 50

	nudgeFactor := 100.0 / 48000.0

	// TODO: give names to magic constants we have here

	L := int(Δt)

	tmp := make([]float32, L*2)

	sfx.Render(scene, camera, t0, tmp, 2, sampleRate)

	nudge := queueingTargetSamples - au().Queued()/(2*4)

	Lnudged := L + max(int(math.Ceil(float64(nudge)*nudgeFactor)), -L/2)

	resamplingRatio := float64(L) / float64(Lnudged)

	tmp2 := make([]float32, Lnudged*2)

	resample(tmp2, tmp, 2, resamplingRatio)

	au().Write(unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(tmp2))), len(tmp2)*4))
}

func resample(dst []float32, src []float32, channels int, ratio float64) {
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
	if 0 <= i && i < len(x) {
		return x[i]
	}
	return *new(T)
}
