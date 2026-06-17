package main

import (
	"log"
	"sync"

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

// TODO: some kind of mechanism to enqueue uisounds and do bgm/soundtrack? I
// guess we should just use our 3d sound renderer for this purpose.
