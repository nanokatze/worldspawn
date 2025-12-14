package game

import (
	"time"

	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
)

type Weapon2 interface {
	// Only call this on the server
	CreateGeometry(scene *Scene) ecs.ID
}

type Weapon2GenericProjectileLauncher struct {
	CycleDuration time.Duration `json:",format:units"`

	NextAttack Time
}

var _ Weapon2 = Weapon2GenericProjectileLauncher{}

func (weapon Weapon2GenericProjectileLauncher) CreateGeometry(scene *Scene) ecs.ID {
	root := scene.CreateEntity()
	scene.TranslationRotation.Store(root, TranslationRotation{
		Translation: geometry.DVec3{0.2, 0.4, -0.275},
		Rotation:    geometry.Rot3One(),
	})
	scene.Scale.Store(root, geometry.Vec3Broadcast(1))
	scene.RenderingGeometry.Store(root, PackGeometry(Geometry{Kind: GeometryFileBacked, Filename: "weapons/grenade_launcher/geometries/Grenade_Launcher"}))

	return root
}

// TODO: pass buttons but buttons should be weapon-specific
// TODO: have UpdateSubtick return an object with functions to call on various
// entities to e.g. apply animation etc?
func (weapon Weapon2GenericProjectileLauncher) UpdateSubtick(scene *Scene, operatorID, weaponID ecs.ID, pos geometry.DVec3, rot geometry.Rot3) {

}
