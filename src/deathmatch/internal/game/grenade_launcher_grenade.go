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

func (grenade GrenadeLauncherGrenade) UpdateAfterPhysics(w *Scene, id ecs.Entity, info *UpdateParams) {
	creationTime, _ := w.CreationTime.Get(id)
	explosionTime := creationTime.Add(grenade.Fuse)
	if w.Now.Before(explosionTime) {
		return
	}

	if !info.Speculating {
		effect := w.CreateEntity()
		positionRotation, _ := w.TranslationRotation.Get(id)
		w.TranslationRotation.Set(effect, positionRotation)
		w.SoundEffect.Set(effect, SoundEmitter{
			Effect:   "later.wav",
			PlayTime: w.Now.Add(info.Δt),
		})
		w.DeleteAfter.Set(effect, w.Now.Add(2*time.Second))
	}

	w.Delete.Set(id, struct{}{})
}
