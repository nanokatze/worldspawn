package game

import (
	"worldspawn/geometry-go"
	"worldspawn/internal/ecs"
)

type Weapon2 interface {
	// Only call this on the server
	CreateGeometry(s *Scene) ecs.ID

	// Weapon2UpdateSubtick(s *Scene, operator ecs.ID)
}

type Weapon2GenericProjectileLauncher struct {
}

var _ Weapon2 = Weapon2GenericProjectileLauncher{}

func (w Weapon2GenericProjectileLauncher) CreateGeometry(s *Scene) ecs.ID {
	root := s.CreateEntity()
	s.TranslationRotation.Store(root, TranslationRotation{
		Translation: geometry.DVec3{0.2, 0.4, -0.275},
		Rotation:    geometry.Rot3One(),
	})
	s.Scale.Store(root, geometry.Vec3Broadcast(1))
	s.RenderingGeometry.Store(root, PackGeometry(Geometry{Kind: GeometryFileBacked, Filename: "weapons/grenade_launcher/geometries/Grenade_Launcher"}))

	return root
}
