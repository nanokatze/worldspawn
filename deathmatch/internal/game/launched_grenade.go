package game

import (
	"time"

	"worldspawn/internal/ecs"
)

var grenadeStats = struct {
	ExplosionSound string

	FuseDuration time.Duration `json:",format:iso8601"`
}{
	ExplosionSound: "later.wav",

	FuseDuration: 1400 * time.Millisecond,
}

type LaunchedGrenade struct{}

func (LaunchedGrenade) entity() {}

func (grenade LaunchedGrenade) TimerExpired(w *Scene, grenadeID ecs.ID, info *UpdateParams) {
	grenadeT := w.GetGlobalTransform(grenadeID).TRS()

	if !info.Speculating {
		eff := w.CreateEntity(info)
		w.SetTransform(eff, grenadeT)
		w.Timer.Set(eff, w.Now.Add(2*time.Second))
		w.Entity.Set(eff, DeleteAfter{})
		w.SoundEffect.Set(eff, SoundEmitter{
			Effect:      grenadeStats.ExplosionSound,
			Attenuation: 1,
			PlayTime:    w.Now.Add(info.Δt),
		})
	}

	w.Delete.Set(grenadeID, struct{}{})
}
