package main

import (
	"log"
	"math"
	"sync"
	"time"
	"unsafe"

	sfx "worldspawn/fuckwwise"
	"worldspawn/fuckwwise/interpolators"
	"worldspawn/sdl"
)

const sampleRate = 48000

var au *sdl.AudioStream

func initAudio() {
	sdl.InitSubSystem(sdl.INIT_AUDIO)

	// TODO: failures here don't really need to be fatal

	var err error
	au, err = sdl.OpenAudioDeviceStream(
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
}

var sources = make(map[string]*sfx.Source)

var audioDone = new(sync.WaitGroup)

func renderAudio(scene sfx.Scene, t0, Δt time.Duration) {
	// TODO: prevent scheduling audio way too far ahead.

	oldAudioDone := audioDone

	audioDone = new(sync.WaitGroup)
	audioDone.Add(1)

	// TODO: make this tunable at runtime
	queueingTargetSamples := 48000 / 50

	nudgeFactor := 100.0 / 48000.0

	go func() {
		oldAudioDone.Wait()

		defer audioDone.Done()

		// TODO: give names to magic constants we have here

		// NOTE: when sampleRate * Δt doesn't divide by 1e9, we could adjust L,
		// t0 and Δt slightly between each render call, to compensate for error.
		// This is rather annoying to deal with so let's rather just not handle
		// this case for now and assert that it divides cleanly.
		L := int(int64(sampleRate) * int64(Δt) / 1e9)

		tmp := make([]float32, L*2)

		sfx.Render(&scene, t0, tmp, 2, sampleRate)

		// TODO: try low pass filtering nudge with 0.5 weight?
		nudge := queueingTargetSamples - au.Queued()/(2*4)

		Lnudged := L + max(int(math.Ceil(float64(nudge)*nudgeFactor)), -L/2)

		resamplingRatio := float64(L) / float64(Lnudged)

		tmp2 := make([]float32, Lnudged*2)

		resample(tmp2, tmp, 2, resamplingRatio)

		au.Write(unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(tmp2))), len(tmp2)*4))
	}()
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
