package game

import (
	"worldspawn/geometry-go"
)

type CosmeticOffset struct {
	Offset    geometry.Vec3
	StartTime Time
	EndTime   Time
}

func (cosmeticOffset CosmeticOffset) Evaluate(now Time) geometry.Vec3 {
	// TODO: rename
	x := durationToFloatSeconds(cosmeticOffset.EndTime.Sub(now)) /
		durationToFloatSeconds(cosmeticOffset.EndTime.Sub(cosmeticOffset.StartTime))
	xClamped := min(max(x, 0), 1)

	return cosmeticOffset.Offset.Scale(float32(xClamped))
}

// TODO: let us do a transform here?
// TODO: make RenderingGeometry be just a blob string with special json and/or
// text de/serialization routines. Having it be a blob string would mean we can
// hash a otherwise possibly complicated rendering geometry straightforwardly.
type RenderingGeometry struct {
	// Kind     string
	Filename string
}

type SoundEffect struct {
	Effect string
	// TODO: we might want to place some (all?) effect arguments into their
	// own component
	PlayTime Time
	// StopTime time.Duration
}
