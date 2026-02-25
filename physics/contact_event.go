package physics

import "worldspawn/internal/gmath"

type ContactEvent struct {
	Type         int32
	Body1, Body2 PerBodyContactData
	Normal       gmath.Vec3 // contact normal
}

type PerBodyContactData struct {
	BodyID          BodyID // TODO: rename to BodyID
	SubShapeID      uint32
	Active          bool
	Position        gmath.DVec3
	Rotation        gmath.Vec3
	LinearVelocity  gmath.Vec3
	AngularVelocity gmath.Vec3
}

func (ce ContactEvent) SwapBodies() ContactEvent {
	return ContactEvent{
		Type:   ce.Type,
		Body1:  ce.Body2,
		Body2:  ce.Body1,
		Normal: ce.Normal.Scale(-1),
	}
}
