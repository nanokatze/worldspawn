package game

import (
	"math"
	"time"

	"worldspawn/internal/ecs"
)

// TODO: instead of "Launched" use "InFlight"

var grenadeStats = struct {
	ExplosionSound string

	FuseDuration time.Duration `json:",format:iso8601"`
}{
	ExplosionSound: "explosion.wav",

	FuseDuration: 1400 * time.Millisecond,
}

type InFlightGrenade struct{}

func (InFlightGrenade) entity() {}

var _ interface {
	Thinker
	Thinker2
} = InFlightGrenade{}

// TODO: rename this message
type Explode struct{}

func (grenade InFlightGrenade) Think(w *Scene, id ecs.ID, info *UpdateParams) {
	T := w.GetTransform(id).Compose()

	w.doExplosion(
		Impact{
			Type:      0,
			Damage:    300,
			Inflictor: id, // TODO: this should be the character, actually
		},
		T,
		sphericalExplosion, 3,
		4*math.Pi/500)

	w.SendMessage(id, Explode{})
}

func (grenade InFlightGrenade) HandleMessage(w *Scene, id ecs.ID, msg any, info *UpdateParams) {
	switch msg := msg.(type) {
	case Explode:
		_ = msg

		T := w.GetTransform(id)

		w.ClearEntity(id)
		w.SetTransform(id, T)
		w.NextThink.Set(id, w.Now.Add(2*time.Second))
		w.Entity.Set(id, DeleteAfter{})
		w.SoundEffect.Set(id, SoundEmitter{
			Effect:      grenadeStats.ExplosionSound,
			Attenuation: 1,
			PlayTime:    w.Now.Add(info.Δt),
		})
	}
}
