package physics

import "worldspawn/internal/gmath"

type ContactEvent struct {
	Type         int32
	Body1, Body2 PerBodyContactData
	Normal       gmath.Vec3f32 // contact normal
}

type PerBodyContactData struct {
	BodyID          BodyID // TODO: rename to BodyID
	SubShapeID      uint32
	Active          bool
	Position        gmath.Vec3f64
	Rotation        gmath.Vec3f32
	LinearVelocity  gmath.Vec3f32
	AngularVelocity gmath.Vec3f32
}

func (ce ContactEvent) SwapBodies() ContactEvent {
	return ContactEvent{
		Type:   ce.Type,
		Body1:  ce.Body2,
		Body2:  ce.Body1,
		Normal: ce.Normal.Scale(-1),
	}
}
