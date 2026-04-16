package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"path"
	"sync"
	"unsafe"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/internal/apostprocess"
	"worldspawn/internal/arenderer"
	"worldspawn/internal/fuckwwise/opusfile"
	"worldspawn/internal/fuckwwise/wav"
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

// TODO: rethink wtf is going on here pls
func renderAudio(scene *arenderer.Scene, cameraTransform gmath.Affine3f32, Δt int64) {
	// TODO: make this tunable at runtime
	queueingTargetSamples := 48000 / 50

	nudgeFactor := 100.0 / 48000.0

	// TODO: give names to magic constants we have here

	L := int(Δt)

	tmp := make([]float32, L*2)

	scene.Render(
		arenderer.Film{
			Samples:  tmp,
			Channels: 2,
		},
		cameraTransform)

	nudge := queueingTargetSamples - au().Queued()/(2*4)

	Lnudged := L + max(int(math.Ceil(float64(nudge)*nudgeFactor)), -L/2)

	resamplingRatio := float64(L) / float64(Lnudged)

	tmp2 := make([]float32, Lnudged*2)

	apostprocess.Resample(tmp2, tmp, 2, resamplingRatio)

	au().Write(unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(tmp2))), len(tmp2)*4))
}

var sources = make(map[string][]float32)

func lookupsound(id string) []float32 {
	effect, ok := sources[id]
	if !ok {
		f, err := game.Data.Open(id)
		if err != nil {
			// TODO: should be non-fatal
			panic(fmt.Sprintf("failed to open file %v", id))
		}

		switch path.Ext(id) {
		case ".wav":
			reader, err := wav.NewReader(f.(io.ReaderAt))
			if err != nil {
				panic(err)
			}
			samples, _ := readSamples(reader, reader.Format())
			effect = extractChannel(samples, reader.Channels(), 0)

		case ".opus":
			reader, _ := opusfile.NewReader(f)
			samples, _ := readSamples(reader, wav.FORMAT_F32)
			effect = extractChannel(samples, reader.Channels(), 0)

		default:
			panic("unsupported")
		}

		sources[id] = effect
	}

	return effect
}
