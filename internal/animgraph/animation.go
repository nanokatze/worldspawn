package animgraph

import "math"

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

// TODO: make the internals private
type Animation struct {
	Frames   int
	Channels map[string]Track
}
