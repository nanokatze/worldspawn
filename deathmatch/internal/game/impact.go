package game

import "worldspawn/internal/ecs"

type ImpactType int

// TODO: it would be nice if we could register these at runtime somehow.
const (
	_ ImpactType = iota
	BlastImpact
	BlastImpactWithFragmentation
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
	// TODO: rename to Attacker?
	// TODO: we might need more attribution fields (weapon id for weapon name
	// and icon, etc)
	Attacker ecs.ID

	Type ImpactType

	Damage int32

	Δv Velocity
}

// TODO: rename?
func (impact Impact) Apply(world *World, id ecs.ID, updateParams *UpdateParams) {
	scriptFuncs := world.GetScriptFuncs(id)
	if scriptFuncs.Impact != nil {
		scriptFuncs.Impact(world, id, impact, updateParams)
	}

	// TODO: only do this if Impact is not implemented?
	if vel, ok := world.Velocity.Get(id); ok {
		world.Velocity.Set(id, vel.Add(impact.Δv))
	}
}
