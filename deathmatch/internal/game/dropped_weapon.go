package game

import (
	"log"
	"slices"

	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
)

type DroppedWeapon struct {
	Weapon ecs.ID
}

func (DroppedWeapon) entity() {}

// TODO: this probs should be global? and moved to dropped_weapon.go.......
// TODO: use weaponState in place of weapon?
func (w *Scene) CreateDroppedWeapon(weaponID ecs.ID, info *UpdateParams) ecs.ID {
	weapon := mustOk(SceneGetEntity[Weapon](w, weaponID))

	dropped := w.CreateEntity(info)
	w.SetTransform(dropped, gmath.TRS3One[float64]())
	w.Entity.Set(dropped, DroppedWeapon{Weapon: weaponID})

	weapon.WeaponCreateGeometry(w, dropped, info)

	w.SetParent(weaponID, dropped)

	return dropped
}

// TODO: move handling of this into Character
func (dropped DroppedWeapon) UpdateBeforePhysics(w *Scene, ourID ecs.ID, info *UpdateParams) {
	trsOurs := mustOk(w.GetGlobalTransform(ourID))

	for playerID, entity := range ecs.All(&w.Entity) {
		if character, ok := entity.(Character); ok {
			trsPlayer := mustOk(w.GetGlobalTransform(playerID))

			if trsOurs.T.Sub(trsPlayer.T).Length() <= 1.1 {
				freeSlot := slices.Index(character.Slots[:], 0)
				if freeSlot != -1 {
					// TODO: properly factor this out
					character.Slots[freeSlot] = dropped.Weapon
					w.Entity.Set(playerID, character)
					w.SetParent(dropped.Weapon, playerID)
					w.Delete.Set(ourID, struct{}{})
					log.Printf("gave weapon %v to the player %v", dropped.Weapon, playerID)
					return
				}
			}
		}
	}
}
