package game

import (
	"time"

	"worldspawn/internal/ecs"
)

type GrenadeLauncherGrenade struct {
	Fuse time.Duration `json:",format:iso8601"`

	// TODO: distance impact multiplier parameters
}

var _ UpdateAfterPhysics = GrenadeLauncherGrenade{}

func (grenade GrenadeLauncherGrenade) UpdateAfterPhysics(w *Scene, id ecs.ID, info *UpdateParams) {
	creationTime, _ := w.CreationTime.Get(id)
	explosionTime := creationTime.Add(grenade.Fuse)
	if explosionTime.After(w.Now) {
		return
	}

	if !info.Speculating {
		effect := w.CreateEntity(info)
		trs, _ := w.GetGlobalTRS(id)
		w.SetGlobalTRS(effect, trs)
		w.SoundEffect.Set(effect, SoundEmitter{
			Effect:   "later.wav",
			PlayTime: w.Now.Add(info.Δt),
		})
		w.DeleteAfter.Set(effect, w.Now.Add(2*time.Second))
	}

	w.Delete.Set(id, struct{}{})
}
