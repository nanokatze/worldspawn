package game

import (
	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
)

// TODO: give this a better name
type Visibility struct {
	Mask uint8
	// TODO: replace with an arbitrary int64 id so we can have the same cameras
	// share visibility sets? Or should this be a set of ids/ecs.IDs?
	Camera ecs.ID
}

type CosmeticOffset struct {
	Offset geometry.Vec3
	T0     Time
	T1     Time // replace with time.Duration?
}

func (cosmeticOffset CosmeticOffset) Eval(now Time) geometry.Vec3 {
	// TODO: rename
	x := durationToFloatSeconds(cosmeticOffset.T1.Sub(now)) /
		durationToFloatSeconds(cosmeticOffset.T1.Sub(cosmeticOffset.T0))
	xClamped := min(max(x, 0), 1)

	return cosmeticOffset.Offset.Scale(float32(xClamped))
}

// TODO: replace with string identifying the effect and a bag of state for that
// effect. Alternatively we could use an interface for now, which would be the
// better option.
type SoundEmitter struct {
	Effect      string
	Attenuation float32
	PlayTime    Time
}

type SoundEffect string
