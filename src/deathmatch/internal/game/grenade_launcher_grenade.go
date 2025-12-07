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

func (grenade GrenadeLauncherGrenade) UpdateAfterPhysics(w *Scene, id ecs.ID, info *UpdateParams) {
	creationTime, _ := w.CreationTime.Load(id)
	explosionTime := creationTime.Add(grenade.Fuse)
	if w.Now.Before(explosionTime) {
		return
	}

	if !info.Speculating {
		effect := w.CreateEntity()
		positionRotation, _ := w.TranslationRotation.Load(id)
		w.TranslationRotation.Store(effect, positionRotation)
		w.SoundEffect.Store(effect, SoundEmitter{
			Effect:   "later.wav",
			PlayTime: w.Now.Add(info.Δt),
		})
		w.DeleteAfter.Store(effect, w.Now.Add(2*time.Second))
	}

	w.Delete.Store(id, struct{}{})
}
