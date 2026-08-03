package game

import (
	"io"
	"log"
	"unique"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
	"worldspawn/internal/loaders/audio"
	"worldspawn/internal/loaders/audio/wav"
)

// TODO: make it more general
type VisibilityCondition struct {
	Mask uint8
	// TODO: replace with an arbitrary int64 id so we can have the same cameras
	// share visibility sets? Or should this be a set of ids/ecs.IDs?
	Camera ecs.ID
}

// TODO: make it more general and implement viewshake through this?
type CosmeticOffset struct {
	Alpha  float32
	T0     Time
	Offset gmath.Vec3f32
}

func (cosmeticOffset CosmeticOffset) Eval(now Time) gmath.Vec3f32 {
	extinction := 1.0 / (1 + float64(cosmeticOffset.Alpha)*durationToFloatSeconds(max(now.Sub(cosmeticOffset.T0), 0)))
	return cosmeticOffset.Offset.Scale(float32(extinction))
}

type SoundEmitter struct {
	Effect      unique.Handle[string]
	Attenuation float32
	PlayTime    Time // TODO: we also need to equip it with a sample offset
}

type LoopedSound struct {
	Sound           unique.Handle[string]
	Attenuation     float32
	LengthInSamples int64 // TODO: make this private and non-txable?
}

// Ugly, we should init() on update or something like that
func (a *LoopedSound) Init() {
	// TODO: factor this out into a function in fuckwwise probs

	f, err := Data.Open(a.Sound.Value())
	if err != nil {
		log.Print("got an error while trying to init LoopedSound: ", err)
		return
	}
	defer f.Close()

	wr, err := audio.NewReader(f.(io.ReaderAt))
	if err != nil {
		log.Print("got an error while trying to init LoopedSound: ", err)
		return
	}

	config := wr.Config()

	if config.Channels != 1 {
		panic("only 1 channel is supported")
	}

	off, err := wr.(io.ReadSeeker).Seek(0, io.SeekEnd)
	if err != nil {
		log.Print("got an error while trying to init LoopedSound: ", err)
		return
	}

	var siz int
	switch wav.Format(config.Format) {
	case wav.FORMAT_S16:
		siz = 2
	case wav.FORMAT_F32:
		siz = 4
	default:
		panic("unreachable")
	}

	a.LengthInSamples = off / int64(config.Channels*siz)
}

func (e Entity) SetVisibilityCondition(v VisibilityCondition) {
	e.world.VisibilityCondition.Store(e.id.Index(), v)
}

func (e Entity) SetCosmeticOffset(v CosmeticOffset) { e.world.CosmeticOffset.Store(e.id.Index(), v) }

func (e Entity) SetRenderingGeometry(v unique.Handle[string]) {
	e.world.RenderingGeometry.Store(e.id.Index(), v)
}

func (e Entity) SetSoundEffect(v SoundEmitter) { e.world.SoundEffect.Store(e.id.Index(), v) }
