package animation

import (
	"encoding/json/v2"
	"io"
	"slices"
	"time"
)

type addressMode int8

const (
	_ addressMode = iota
	addressModeClamp
	addressModeRepeat
)

// TODO: address mode. We need to know what to report outside of the defined range, either clamp or repeat
type Animation struct {
	frameRate   rat128
	addressMode addressMode
	duration    int
	channels    []string
	data        [][]float32
}

func SampleTime(a *Animation, t time.Duration, v []float32) {
	frameRate := a.FrameRate()
	// TODO: avoid overflow
	a.Sample(int64(t)*frameRate[0]/frameRate[1], v)
}

func SampleNormalized(a *Animation, t float32, v []float32) {
	a.Sample(int64(t*float32(a.Duration())*1e9), v)
}

// Frame rate of the animation, as ratio of two integers.
func (a *Animation) FrameRate() [2]int64 { return [2]int64{a.frameRate.a, a.frameRate.b} }

// Duration of the animation, in frames. If the animation is periodic, this is
// the duration of the periodic segment.
func (a *Animation) Duration() int { return a.duration }

func (a *Animation) Channels() []string { return a.channels }

// Take a sample at time t, in frames. Most users will want to use SampleTime or
// SampleNormalized functions instead of this.
//
// TODO: what do we do if we don't want all of the channels?
// TODO: a convenience variant that samples a single channel?
// TODO: use a less annoying way to specify t?
func (a *Animation) Sample(t int64, v []float32) {
	f0 := int(t / 1e9)
	f1 := int((t + 1e9 - 1) / 1e9)

	// TODO: factor address mode handling out
	switch a.addressMode {
	case addressModeClamp:
		f0 = min(max(f0, 0), a.duration-1)
		f1 = min(max(f1, 0), a.duration-1)

	case addressModeRepeat:
		// TODO: handle the case when either f0 or f1 are negative.
		f0 = f0 % a.duration
		f1 = f1 % a.duration

	default:
		panic("unreachable")
	}

	uhh := float32(t%1e9) / 1e9

	for i, ch := range a.data {
		v[i] = ch[f0]*(1-uhh) + ch[f1]*uhh
	}
}

// TODO: this should be implemented by various loaders
func Read(r io.Reader) (*Animation, error) {
	var tmp struct {
		FrameRate   rat128
		AddressMode string
		Duration    int
		Channels    []struct {
			Name string
			Data []float32
		}
	}
	if err := json.UnmarshalRead(r, &tmp, json.StringifyNumbers(true)); err != nil {
		return nil, err
	}

	// TODO: validate that tmp.FrameRate.b <= 1001

	return &Animation{
		frameRate:   tmp.FrameRate,
		addressMode: addressModeClamp,
		duration:    tmp.Duration,
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
