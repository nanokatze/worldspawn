package game

import (
	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
)

type Visibility struct {
	Mode   int8 // 1=viewmodel, 2=worldmodel; TODO: make a enum
	Camera ecs.ID
	// TODO: replace with an arbitrary int64 id so we can have the same cameras
	// share visibility sets? Or should this be a set of ids/ecs.IDs?
}

type CosmeticOffset struct {
	Offset    geometry.Vec3
	StartTime Time
	EndTime   Time // TODO: make this duration instead, should improve compression
}

func (cosmeticOffset CosmeticOffset) Eval(now Time) geometry.Vec3 {
	// TODO: rename
	x := durationToFloatSeconds(cosmeticOffset.EndTime.Sub(now)) /
		durationToFloatSeconds(cosmeticOffset.EndTime.Sub(cosmeticOffset.StartTime))
	xClamped := min(max(x, 0), 1)

	return cosmeticOffset.Offset.Scale(float32(xClamped))
}

type SoundEmitter struct {
	Effect      string
	Attenuation float32
	PlayTime    Time
}
