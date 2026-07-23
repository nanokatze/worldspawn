package animation

import (
	"encoding/json/v2"
	"io"
	"math"
	"slices"
	"unique"
)

// TODO: use time.Duration instead of float64 please?

type Animation struct {
	frames   int
	channels []unique.Handle[string]
	data     [][]float32
}

func (a *Animation) Channels() []unique.Handle[string] { return a.channels }

// TODO: what do we do if we don't want all of the channels?
// TODO: a convenience variant that samples a single channel
func (a *Animation) Sample(t float64, out []float32) {
	s0 := int(math.Floor(t))
	s1 := int(math.Ceil(t))
	if !(0 <= s0 && s1 < a.frames) {
		clear(out)
		return
	}

	uhh := float32(t - math.Floor(t))

	for i, ch := range a.data {
		out[i] = ch[s0]*(1-uhh) + ch[s1]*uhh
	}
}

// TODO: this should be implemented by various loaders
func Read(r io.Reader) (*Animation, error) {
	var tmp struct {
		Frames   int
		Channels []struct {
			Name string
			Data []float32
		}
	}
	if err := json.UnmarshalRead(r, &tmp, json.StringifyNumbers(true)); err != nil {
		return nil, err
	}

	return &Animation{
		frames: tmp.Frames,
		channels: slices.Collect(func(yield func(unique.Handle[string]) bool) {
			for _, ch := range tmp.Channels {
				yield(unique.Make(ch.Name))
			}
		}),
		data: slices.Collect(func(yield func([]float32) bool) {
			for _, ch := range tmp.Channels {
				yield(ch.Data)
			}
		}),
	}, nil
}
