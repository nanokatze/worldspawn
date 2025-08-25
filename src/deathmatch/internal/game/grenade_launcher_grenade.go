package game

import (
	"time"

	"worldspawn/internal/ecs"
)

type GrenadeLauncherGrenade struct {
	Fuse time.Duration `json:",format:units"`

	// TODO: distance impact multiplier parameters
}

var _ UpdateAfterPhysics = GrenadeLauncherGrenade{}

func init() {
	registerEntity[GrenadeLauncherGrenade]()
}

func (grenade GrenadeLauncherGrenade) UpdateAfterPhysics(w *Scene, id ecs.ID, info *UpdateParams) {
	spawnTime, _ := w.CreatedAt.Load(id)

	if w.Now < spawnTime.Add(grenade.Fuse) {
		return
	}

	if !info.Speculating {
		effect := w.CreateEntity()
		positionRotation, _ := w.TranslationRotation.Load(id)
		w.TranslationRotation.Store(effect, positionRotation)
		w.SoundEffect.Store(effect, SoundEffect{
			Effect:   "later.wav",
			PlayTime: w.Now.Add(info.Δt),
		})
		w.DeleteAfter.Store(effect, w.Now.Add(2*time.Second))
	}

	w.Delete.Store(id, struct{}{})
}
