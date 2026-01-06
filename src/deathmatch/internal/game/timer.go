package game

import "worldspawn/internal/ecs"

func (w *Scene) processTimers(updateParams *UpdateParams) {
	// TODO: assert that these entities implement TimerExpired interface, rather
	// than doing a join over them.
	for id, v := range ecs.Join(&w.Timer, &w.Entity) {
		if v.V1.After(w.Now) {
			continue
		}

		timerExpired, ok := SceneGetEntity[interface {
			Entity
			TimerExpired(w *Scene, id ecs.ID, info *UpdateParams)
		}](w, id)
		if ok {
			timerExpired.TimerExpired(w, id, updateParams)
		}
	}
}
