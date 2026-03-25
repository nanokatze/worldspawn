package physics

import "worldspawn/internal/gmath"

type Scene struct {
}

// TODO: setting body states and stuff. I guess we should have separate methods for all the fields?

func (s *Scene) SetGravity(gravity gmath.Vec3f32) {
}

func (s *Scene) UpdateAccel() {
}

func (s *Scene) Update(dt float32) {
}

// Checklist:
// Setting and getting body info, deleting bodies
// A method to rebuild the acceleration structure
// Iteration over bodies that were active after an update
// Intersection queries
// Shapes (must have materials)
