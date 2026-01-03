package game

import (
	"time"

	"worldspawn/internal/ecs"
)

var grenadeStats = struct {
	FuseDuration   time.Duration `json:",format:iso8601"`
	ExplosionSound string
}{
	FuseDuration:   1400 * time.Millisecond,
	ExplosionSound: "later.wav",
}

type LaunchedGrenade struct{}

func (LaunchedGrenade) entity() {}

var _ UpdateAfterPhysics = LaunchedGrenade{}

func (grenade LaunchedGrenade) UpdateAfterPhysics(w *Scene, id ecs.ID, info *UpdateParams) {
	creationTime, _ := w.CreationTime.Get(id)
	explosionTime := creationTime.Add(grenadeStats.FuseDuration)
	if explosionTime.After(w.Now) {
		return
	}

	if !info.Speculating {
		effect := w.CreateEntity(info)
		trs, _ := w.GetGlobalTRS(id)
		w.SetGlobalTRS(effect, trs)
		w.SoundEffect.Set(effect, SoundEmitter{
			Effect:      grenadeStats.ExplosionSound,
			Attenuation: 1,
			PlayTime:    w.Now.Add(info.Δt),
		})
		w.DeleteAfter.Set(effect, w.Now.Add(2*time.Second))
	}

	w.Delete.Set(id, struct{}{})
}
