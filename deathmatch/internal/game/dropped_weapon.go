package game

import (
	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

type DroppedWeapon struct {
	Weapon ecs.ID
}

func (DroppedWeapon) entity() {}

func (w *Scene) CreateDroppedWeapon(weaponID ecs.ID, info *UpdateParams) ecs.ID {
	weapon := mustOk(SceneGetEntity[Weapon](w, weaponID))

	dropped := w.CreateEntity(info)
	w.SetTransform(dropped, gmath.TRS3One[float64]())
	w.Entity.Set(dropped, DroppedWeapon{Weapon: weaponID})

	w.SetParent(weapon.CreateProp(w, info), dropped)

	w.SetParent(weaponID, dropped)

	return dropped
}

// TODO: move handling of this into Gladiator entirely
func (dropped DroppedWeapon) PrePhysicsStep(w *Scene, ourID ecs.ID, info *UpdateParams) {
	T := w.GetGlobalTransform(ourID)

	for playerID, entity := range ecs.All(&w.Entity) {
		if _, ok := entity.(Gladiator); ok {
			playerT := w.GetGlobalTransform(playerID)

			if T.T.Sub(playerT.T).Length() <= 1.1 {
				w.GiveWeapon(playerID, dropped.Weapon)
				w.Delete.Set(ourID, struct{}{})
			}
		}
	}
}
