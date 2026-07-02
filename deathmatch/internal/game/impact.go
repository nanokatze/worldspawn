package game

import "worldspawn/internal/ecs"

type ImpactType int

// TODO: it would be nice if we could register these at runtime somehow.
const (
	_ ImpactType = iota
	BlastImpact
	BlastImpactWithFragmentation
	BlastImpactWithNoDamage // TODO: implement this
	BulletImpact
)

var impactBleedFactor = map[ImpactType]float32{
	BlastImpactWithFragmentation: 0.33,
	BulletImpact:                 0.5,
}

// TODO: make this more structured and possibly pull this from json
var impactForceFactor = map[ImpactType]float32{
	BlastImpact:                  0.2,
	BlastImpactWithFragmentation: 0.2,
}

// TODO: reorder the fields in here
type Impact struct {
	// character this should be attributed to
	//
	// TODO: we might need more attribution fields (weapon id for weapon name
	// and icon, etc)
	Attacker ecs.ID

	Type ImpactType

	Damage int32

	Δv Velocity
}

// TODO: kill this in favor of enqueueImpact
func (impact Impact) Apply(info *UpdateParams, id ecs.ID, io IO) {
	scriptFuncs := io.world.GetScriptFuncs(id)
	if scriptFuncs.Impact != nil {
		scriptFuncs.Impact(info, id, impact, io)
	}

	// TODO: only do this if Impact is not implemented and make entities
	// implementing Impact do everything otherwise?
	if vel, ok := io.world.Velocity.Get(id); ok {
		io.world.Velocity.Set(id, vel.Add(impact.Δv))
	}
}

/*
func enqueueImpact(io *IO, target ecs.ID, impact Impact) {

}
*/
