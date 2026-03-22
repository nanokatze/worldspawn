package animgraph

// TODO: make the internals private
type Animation struct {
	// TODO: rename to tracks?
	Frames   int
	Channels map[string][]float32
}

// TODO: make Sample write into a huge "point" object
func (a *Animation) Sample(channel string, t int) float32 {
	track := a.Channels[channel]
	if !(0 <= t && t < len(track)) {
		return 0
	}
	return track[t]
}
