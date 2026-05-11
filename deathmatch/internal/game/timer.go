package game

import (
	"fmt"

	"worldspawn/internal/ecs"
)

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

		timer, ok := SceneGetEntity[Timer](w, id)
		if !ok {
			panic(fmt.Sprintf("timer expired on an object that does not implement the interface id=%d", id))
			continue
		}
		timer.TimerExpired(w, id, updateParams)
	}
}
