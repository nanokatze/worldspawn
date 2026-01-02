package game

import (
	"io"
	"log"
	"time"

	"worldspawn/internal/ecs"
	sfx "worldspawn/internal/fuckwwise"
	"worldspawn/internal/fuckwwise/wav"
)

// TODO: add ability to randomize the repeating segments

// TODO: should be embeddable/reusable with other entities or be made into a
// component, maybe significantly generalzed in that case
// TODO: rename to LoopingSound?
type LoopedSound struct {
	Sound           string
	LengthInSamples int64 // TODO: make this private and non-txable?
}

func (LoopedSound) entity() {}

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

func (a LoopedSound) UpdateBeforePhysics(scene *Scene, id ecs.ID, info *UpdateParams) {
	// TODO: make repeat sample-perfect

	soundEffect, _ := scene.SoundEffect.Get(id)
	if soundEffect.PlayTime.Add(time.Duration(a.LengthInSamples * 1e9 / 48000)).After(scene.Now) {
		return
	}

	soundEffect.Effect = a.Sound
	soundEffect.PlayTime = scene.Now
	scene.SoundEffect.Set(id, soundEffect)
}
