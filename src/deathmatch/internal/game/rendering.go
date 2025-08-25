package game

import (
	"strings"
	"worldspawn/geometry-go"
	"worldspawn/internal/nice"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
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

type RenderingGeometry2 string

// TODO: rename
func (tmp *RenderingGeometry) Unpack(asd RenderingGeometry2) {
	if err := nice.UnmarshalDecode(nice.NewDecoder(strings.NewReader(string(asd))), tmp); err != nil {
		panic(err)
	}
}

// TODO: rename
func (tmp RenderingGeometry) Pack() RenderingGeometry2 {
	var buf strings.Builder
	if err := nice.MarshalEncode(nice.NewEncoder(&buf), &tmp); err != nil {
		panic(err)
	}
	return RenderingGeometry2(buf.String())
}

func (geo *RenderingGeometry2) UnmarshalJSONFrom(d *jsontext.Decoder) error {
	var tmp RenderingGeometry
	if err := json.UnmarshalDecode(d, &tmp); err != nil {
		return err
	}
	*geo = tmp.Pack()
	return nil
}

type SoundEffect struct {
	Effect string
	// TODO: we might want to place some (all?) effect arguments into their
	// own component
	PlayTime Time
	// StopTime time.Duration
}
