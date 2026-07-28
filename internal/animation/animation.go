package animation

import (
	"encoding/json/v2"
	"io"
	"slices"
	"time"
)

// TODO: address mode. We need to know what to report outside of the defined range, either clamp or repeat
type Animation struct {
	addressMode bool // true=repeat; false=clamp; TODO: make a proper enum for it
	frames      int
	frameRate   int
	channels    []string
	data        [][]float32
}

func SampleTime(a *Animation, t time.Duration, out []float32) {
	a.Sample(int64(t)*int64(a.FrameRate()), out)
}

func SampleNormalized(a *Animation, t float32, out []float32) {
	a.Sample(int64(t*float32(a.Frames())*1e9), out)
}

// Duration of the animation. If an animation is periodic, this is the duration
// of the periodic segment.
func (a *Animation) Frames() int { return a.frames }

// Frame rate of the animation.
// TODO: allow this to be fractional (UGHHHHHHHHH)
func (a *Animation) FrameRate() int { return a.frameRate }

func (a *Animation) Channels() []string { return a.channels }

// Take a sample at time t, in frames. Most users will want to use SampleTime or
// SampleNormalized functions instead of this.
//
// TODO: what do we do if we don't want all of the channels?
// TODO: a convenience variant that samples a single channel?
// TODO: use a less annoying way to specify t?
func (a *Animation) Sample(t int64, out []float32) {
	f0 := int(t / 1e9)
	f1 := int((t + 1e9 - 1) / 1e9)

	// TODO: factor address mode handling out
	switch a.addressMode {
	case false: // clamp
		f0 = min(max(f0, 0), a.frames-1)
		f1 = min(max(f1, 0), a.frames-1)

	case true: // repeat
		// TODO: handle the case when either f0 or f1 are negative.
		f0 = f0 % a.frames
		f1 = f1 % a.frames
	}

	uhh := float32(t%1e9) / 1e9

	for i, ch := range a.data {
		out[i] = ch[f0]*(1-uhh) + ch[f1]*uhh
	}
}

// TODO: this should be implemented by various loaders
func Read(r io.Reader) (*Animation, error) {
	var tmp struct {
		Frames    int
		FrameRate int
		Channels  []struct {
			Name string
			Data []float32
		}
	}
	if err := json.UnmarshalRead(r, &tmp, json.StringifyNumbers(true)); err != nil {
		return nil, err
	}

	return &Animation{
		frames:    tmp.Frames,
		frameRate: tmp.FrameRate,
		channels: slices.Collect(func(yield func(string) bool) {
			for _, ch := range tmp.Channels {
				yield(ch.Name)
			}
		}),
		data: slices.Collect(func(yield func([]float32) bool) {
			for _, ch := range tmp.Channels {
				yield(ch.Data)
			}
		}),
	}, nil
}
