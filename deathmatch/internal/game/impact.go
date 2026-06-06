package game

import "worldspawn/internal/ecs"

type ImpactType int

// TODO: make this more structured and possibly pull this from json
var impactTypes = []float32{
	0.2,
}

// TODO: reorder the fields in here
type Impact struct {
	// character this should be attributed to
	//
	// TODO: rename to Attacker?
	// TODO: we might need more attribution fields (weapon id for weapon name
	// and icon, etc)
	Inflictor ecs.ID

	Type ImpactType

	Damage float32

	Δv Velocity
}

// TODO: rename?
func (impact Impact) Apply(scene *Scene, id ecs.ID, updateParams *UpdateParams) {
	scriptFuncs := scene.GetScriptFuncs(id)
	if scriptFuncs.Impact != nil {
		scriptFuncs.Impact(scene, id, impact, updateParams)
	}

	// TODO: only do this if Impact is not implemented?
	if vel, ok := scene.Velocity.Get(id); ok {
		scene.Velocity.Set(id, vel.Add(impact.Δv))
	}
}
