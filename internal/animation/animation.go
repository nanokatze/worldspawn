package animation // TODO: rename to animation

import (
	"encoding/json/v2"
	"io"
	"math"
	"time"
	"unique"
)

// TODO: kill this in favor of animation.Sample
type Track []float32

// TODO: should t stay float64, or be changed to float32 or fixed point? I guess
// we could just use time.Duration as well.
func (track Track) Sample(t float64) float32 {
	s0 := int(math.Floor(t))
	s1 := int(math.Ceil(t))
	if !(0 <= s0 && s1 < len(track)) {
		return 0
	}
	uhh := float32(t - math.Floor(t))
	return track[s0]*(1-uhh) + track[s1]*uhh
}

// TODO: make the internals private. Also should be an interface probably.
type Animation struct {
	Frames   int
	Channels map[string]Track
}

func (a *Animation) Channels_() []unique.Handle[string] {
	panic("not implemented")
}

// TODO: allow the user to pass the mask of channels they're interested in and
// assume out is tightly packed?
// TODO: a convenience variant that samples a single channel
func (a *Animation) Sample(t time.Duration, out []float32) {
	panic("not implemented")
}

// TODO: this should be implemented by various loaders
func Read(r io.Reader) (*Animation, error) {
	var animation Animation
	if err := json.UnmarshalRead(r, &animation, json.StringifyNumbers(true)); err != nil {
		return nil, err
	}
	return &animation, nil
}
