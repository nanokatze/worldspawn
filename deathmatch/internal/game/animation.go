package game

import (
	"math"
	"sync"

	"github.com/go-json-experiment/json"

	"worldspawn/internal/geometry"
)

// TODO: move this stuff into its own package probably

/*
type AnimationSampler struct {
	// repeat or clamp to edge
	AddressMode  int
	Unnormalized bool
}
*/

/*
type AnimationSamples struct {
}

type Animation2 struct {
	Channels map[string]AnimationSamples
}
*/

type Animation struct {
	Armature map[string]geometry.TRS3 // TODO: split off into its own component?
	Action   string
	// Channels   map[string][]geometry.Affine3
	// SampleRate int // TODO: this must live as part of Channels. We separately should have a knob for animation playback speed or animation deadline.
	PlayTime Time

	// TODO: playback speed. We'll want both options: a speed factor as well as
	// deadline by which the animation must complete
}

// TODO: these should be getArmature and getAnimation the way we have getShape
// TODO: probably decorate errors with getArmature: $filename: $err

func loadArmature(filename string) (map[string]geometry.TRS3, error) {
	f, err := Data.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var m map[string]geometry.TRS3
	if err := json.UnmarshalRead(f, &m, JSONOptions); err != nil {
		return nil, err
	}
	return m, nil
}

type Action struct {
	sampleRate int
	// TODO: interleave the channels
	// TODO: see what we can do about types of stuff we're interpolating
	samples map[string][]geometry.TRS3
}

// TODO: Sample should take channel *index* instead of name, and wrapping
// behavior
func (a *Action) Sample(t float64, channel string) geometry.TRS3 {
	samples := a.samples[channel]

	t *= float64(a.sampleRate)

	i := min(max(int(math.Floor(t)), 0), len(samples)-1)
	j := min(max(int(math.Ceil(t)), 0), len(samples)-1)

	return samples[i].NLerp(samples[j], float32(t-math.Floor(t)))
}

var actionCacheMu sync.Mutex
var actionCache = make(map[string]*Action)

func getAnimation(filename string) *Action {
	actionCacheMu.Lock()
	action, ok := actionCache[filename]
	actionCacheMu.Unlock()
	if !ok {
		f, err := Data.Open(filename)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		var m map[string][]geometry.TRS3
		if err := json.UnmarshalRead(f, &m, json.StringifyNumbers(true)); err != nil {
			panic(err)
		}

		// TODO: export sample rate to file so we can load it
		sampleRate := 60
		if filename == "anim_test.json" {
			sampleRate = 100
		}

		action = &Action{sampleRate: sampleRate, samples: m}

		actionCacheMu.Lock()
		actionCache[filename] = action
		actionCacheMu.Unlock()
	}
	return action
}
