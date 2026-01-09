package game

import "worldspawn/internal/ecs"

// TODO: rename
type Timer interface {
	Entity

	TimerExpired(w *Scene, id ecs.ID, info *UpdateParams)
}

func (w *Scene) processTimers(updateParams *UpdateParams) {
	for id, deadline := range ecs.All(&w.Timer) {
		if deadline.After(w.Now) {
			continue
		}

		mustOk(SceneGetEntity[Timer](w, id)).TimerExpired(w, id, updateParams)
	}
}
