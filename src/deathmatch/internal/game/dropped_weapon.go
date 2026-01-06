package game

import (
	"log"
	"slices"
	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
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
	w.SetGlobalTRS(dropped, geometry.DTRS3One())
	w.Entity.Set(dropped, DroppedWeapon{Weapon: weaponID})

	weapon.WeaponCreateGeometry(w, dropped, info)

	w.SetParent(weaponID, dropped)

	return dropped
}

// TODO: move handling of this into Character
func (dropped DroppedWeapon) UpdateBeforePhysics(w *Scene, ourID ecs.ID, info *UpdateParams) {
	trsOurs := mustOk(w.GetGlobalTRS(ourID))

	w.SetLocalTRS(ourID, geometry.DTRS3{
		T: trsOurs.T,
		R: geometry.Rot3FromPlaneAngle(geometry.Vec3{0, 0, 1}, float32(float64(w.Now)/1e9)),
		S: geometry.Vec3Broadcast(1),
	})

	for playerID, entity := range ecs.All(&w.Entity) {
		if character, ok := entity.(Character); ok {
			trsPlayer := mustOk(w.GetGlobalTRS(playerID))

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
