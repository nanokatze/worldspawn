package main

import (
	"math"
	"time"
	"unsafe"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/internal/apostprocess"
	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
	"worldspawn/internal/spatialaudio"
)

// Unlike video renderer, audio renderer is tied to ticks

type gameAudioRenderer struct {
	scene *spatialaudio.Scene
}

func (re *gameAudioRenderer) Reset(n int) {
	re.scene = spatialaudio.NewScene(n)
}

// TODO: factor shrinker/stretcher out into apostprocess or something

func (re *gameAudioRenderer) Tick(world *game.World, playerID ecs.ID, t0, t1 game.Time, frameDuration time.Duration) {
	fpsCharacter, _ := game.SceneGetEntity[game.Player](world, playerID)

	camera := fpsCharacter.Camera(world)

	cameraTransform := world.GetGlobalTransform(camera)

	scene := re.scene

	clear(scene.Emitters)

	for id, soundEffect := range ecs.All(&world.SoundEffect) {
		T := world.GetGlobalTransform(id)

		effect := lookupsound(soundEffect.Effect)

		scene.Transform[id.Index()] = gmath.Affine3Convert[float32](T).TRS()

		hmm := min(max(int64(t0.Sub(soundEffect.PlayTime)*48000/1e9), 0), int64(len(effect)))
		scene.Emitters[id.Index()] = effect[hmm:]
	}

	// TODO: rewrite this garbage

	// TODO: make this tunable at runtime
	queueingTargetSamples := 48000 / 50

	nudgeFactor := 100.0 / 48000.0

	// TODO: give names to magic constants we have here

	L := int(t1.Sub(t0) * 48000 / 1e9)

	tmp := make([]float32, L*2)

	scene.Render(
		spatialaudio.Film{
			Samples:  tmp,
			Channels: 2,
		},
		gmath.Affine3Convert[float32](cameraTransform))

	// TODO: this is the place where we could mix other sounds

	nudge := queueingTargetSamples - au().Queued()/(2*4)

	Lnudged := L + max(int(math.Ceil(float64(nudge)*nudgeFactor)), -L/2)

	resamplingRatio := float64(L) / float64(Lnudged)

	tmp2 := make([]float32, Lnudged*2)

	apostprocess.Resample(tmp2, tmp, 2, resamplingRatio)

	au().Write(unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(tmp2))), len(tmp2)*4))
}

func (re *gameAudioRenderer) Subtick(world *game.World, playerID ecs.ID) {}
