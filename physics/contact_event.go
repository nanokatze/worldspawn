package physics

import "worldspawn/geometry-go"

type ContactEvent struct {
	Type         int32
	Body1, Body2 PerBodyContactData
	Normal       geometry.Vec3 // contact normal
}

type PerBodyContactData struct {
	BodyID          BodyID // TODO: rename to BodyID
	SubShapeID      uint32
	Active          bool
	Position        geometry.DVec3
	Rotation        geometry.Vec3
	LinearVelocity  geometry.Vec3
	AngularVelocity geometry.Vec3
}

func (ce ContactEvent) SwapBodies() ContactEvent {
	return ContactEvent{
		Type:   ce.Type,
		Body1:  ce.Body2,
		Body2:  ce.Body1,
		Normal: ce.Normal.Scale(-1),
	}
}
