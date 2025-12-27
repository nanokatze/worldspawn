package game

import (
	"math"
	"time"

	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
)

type Weapon2 interface {
	// Only call this on the server
	CreateGeometry(scene *Scene) ecs.Entity

	// Returns a function that updates the visual. TODO: we also need to return
	// stuff like recoil and such.
	UpdateSubtick(scene *Scene, weaponId, operatorId ecs.Entity, info *UpdateParams) func(ecs.Entity)
}

type Weapon2GenericProjectileLauncher struct {
	CycleDuration time.Duration `json:",format:units"`

	NextAttack Time
}

type Testburger struct {
	BaseColorR float32
	BaseColorG float32
	BaseColorB float32
}

func hsv2rgb(hsv [3]float32) [3]float32 {
	h, s, v := hsv[0], hsv[1], hsv[2]

	c := v * s
	x := c * (1 - float32(math.Abs(math.Mod(float64(h/60), 2)-1)))
	m := v - c
	var tmp [3]float32
	switch {
	case h < 60:
		tmp = [3]float32{c, x, 0}
	case h < 120:
		tmp = [3]float32{x, c, 0}
	case h < 180:
		tmp = [3]float32{0, c, x}
	case h < 240:
		tmp = [3]float32{0, x, c}
	case h < 300:
		tmp = [3]float32{x, 0, c}
	case h < 360:
		tmp = [3]float32{c, 0, x}
	default:
		panic("unreachable")
	}
	tmp[0] += m
	tmp[1] += m
	tmp[2] += m
	return tmp
}

var _ Weapon2 = Weapon2GenericProjectileLauncher{}

// TODO: rename to something else like CreateVisual or CreateRenderingGeometry
func (weapon Weapon2GenericProjectileLauncher) CreateGeometry(scene *Scene) ecs.Entity {
	root := scene.CreateEntity()
	scene.TranslationRotation.Set(root, TranslationRotation{
		Translation: geometry.DVec3{0.2, 0.4, -0.275},
		Rotation:    geometry.Rot3One(),
	})
	scene.Scale.Set(root, geometry.Vec3Broadcast(1))
	scene.RenderingGeometry.Set(root, PackGeometry(Geometry{Kind: GeometryFileBacked, Filename: "weapons/grenade_launcher/geometries/Grenade_Launcher"}))

	return root
}

func (weapon Weapon2GenericProjectileLauncher) UpdateSubtick(scene *Scene, weaponID, operatorID ecs.Entity, info *UpdateParams) func(ecs.Entity) {
	return func(id ecs.Entity) {
		rgb := hsv2rgb([3]float32{
			float32(math.Mod(float64(scene.Now)/1e9*60, 360)),
			1,
			1,
		})
		scene.Entity.Set(id, Testburger{
			BaseColorR: rgb[0],
			BaseColorG: rgb[1],
			BaseColorB: rgb[2],
		})
	}
}
