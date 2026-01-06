package game

import "worldspawn/internal/ecs"

// TODO: rename
type DeleteAfter struct{}

func (DeleteAfter) entity() {}

func (deleteAfter DeleteAfter) TimerExpired(w *Scene, id ecs.ID, info *UpdateParams) {
	w.Delete.Set(id, struct{}{})
}
