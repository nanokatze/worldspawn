package game

import "worldspawn/internal/ecs"

type ImpactType int

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
