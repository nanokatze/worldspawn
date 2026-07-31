package game

type ImpactType int

// TODO: it would be nice if we could register these at runtime
const (
	_ ImpactType = iota
	BlastImpact
	BlastImpactWithFragmentation
	BlastImpactWithNoDamage // TODO: implement this
	BulletImpact
)

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
	Attacker Entity2

	// TODO: we also need to communicate something for the damage/kill icon

	Type ImpactType

	Damage float32

	Δv Velocity
}

// TODO: kill this in favor of enqueueImpact
func (impact Impact) Apply(stx ScriptContext, entity Entity2) {
	scriptFuncs := entity.Script()
	if scriptFuncs.Impact != nil {
		scriptFuncs.Impact(stx, entity, impact)
	}

	// TODO: only do this if Impact is not implemented and make entities
	// implementing Impact do everything otherwise?
	// TODO: gate it behind a good check for if things are being actually
	// simulated, e.g. Motion is Dynamic or some component for receiving
	// velocity on a kinematic body is set.
	entity.SetVelocity(entity.Velocity().Add(impact.Δv))
}

/*
func enqueueImpact(io *IO, target ecs.ID, impact Impact) {

}
*/
