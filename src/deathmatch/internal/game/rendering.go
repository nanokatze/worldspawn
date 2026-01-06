package game

import (
	"io"
	"log"

	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
	sfx "worldspawn/internal/fuckwwise"
	"worldspawn/internal/fuckwwise/wav"
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

// TODO: add ability to randomize the repeating segments

type LoopedSound struct {
	Sound           string
	Attenuation     float32
	LengthInSamples int64 // TODO: make this private and non-txable?
}

// Ugly, we should init() on update or something like that
func (a *LoopedSound) Init() {
	// TODO: factor this out into a function in fuckwwise probs

	f, err := Data.Open(a.Sound)
	if err != nil {
		log.Print("got an error while trying to init LoopedSound: ", err)
		return
	}
	defer f.Close()

	wr, err := wav.NewReader(f.(io.ReaderAt))
	if err != nil {
		log.Print("got an error while trying to init LoopedSound: ", err)
		return
	}

	if wr.Channels() != 1 {
		panic("only 1 channel is supported")
	}

	off, err := wr.Seek(0, io.SeekEnd)
	if err != nil {
		log.Print("got an error while trying to init LoopedSound: ", err)
		return
	}

	var siz int
	switch wr.Format() {
	case sfx.FORMAT_S16:
		siz = 2
	case sfx.FORMAT_F32:
		siz = 4
	default:
		panic("unreachable")
	}

	a.LengthInSamples = off / int64(wr.Channels()*siz)
}
