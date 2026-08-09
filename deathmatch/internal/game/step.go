package game

import (
	"reflect"
	"time"

	"worldspawn/internal/ecs"
)

// TODO: rename to StepContext or something
type UpdateParams struct {
	Δt          time.Duration
	Speculating bool

	Worldspawn
}

// TODO: do not accept updateParams here but raw Δt and flags. We should
// construct UpdateParams ourselves
func (world *World) Step(updateParams UpdateParams) {

	updateParams.Worldspawn = world.Entity(1).ScriptState().(Worldspawn)
	updateParams.Now = updateParams.Now.Add(updateParams.Δt)
	world.Entity(1).SetScriptState(updateParams.Worldspawn)

	world.think(&updateParams)

	world.physicsStep(&updateParams)

	world.handleOutOfBoundsEntities(&updateParams)

	for id, a := range ecs.All(&world.SoundEffectState) {
		soundEffect, _ := world.SoundEffect.Get(id)
		if soundEffect.PlayTime.Add(time.Duration(a.LengthInSamples*1e9/48000)).Compare(updateParams.Now) > 0 {
			continue
		}

		soundEffect.Effect = a.Sound
		soundEffect.Attenuation = a.Attenuation
		soundEffect.PlayTime = updateParams.Now
		world.SoundEffect.Set(id, soundEffect)
	}

	world.deleteMarkedEntities()
}

func (world *World) think(updateParams *UpdateParams) {
	// TODO: optimize the pass over thinkers by having a shadow column

	// Update systems which are allowed to be queried from Think

	world.updatePhysicsShadow(updateParams)

	// Run thinkers

	for id, scriptName := range ecs.All(&world.ScriptState) {
		script := Scripts[reflect.TypeOf(scriptName)]
		if script.Think == nil {
			continue
		}

		// TODO: we'll want a timer wheel of sorts to make this fast
		nextThink, _ := world.NextThink.Get(id)
		if nextThink.Compare(updateParams.Now) > 0 {
			continue
		}

		script.Think(ScriptContext{updateParams, IO{world, uint64(id.Index())}}, world, Entity{world, id})
	}

	// Process the enqueued updates

	world.processUpdates(updateParams)
}

// TODO: make it public so that client replication code can use this? Client
// replication code still needs World.DeleteEntityImmediately. I guess it could
// also do whatever surgery it needs.
func (world *World) deleteMarkedEntities() {
	// Propagate deletion from parents.
	//
	// TODO: make this less gross. We could do a probe whether there's any
	// deletions at all right now.
	{
		var f func(id ecs.ID) bool
		// TODO: we could also rotate this
		f = func(id ecs.ID) bool {
			if id == 0 {
				return false
			}

			if world.Columns.delete.Load(id.Index()) {
				return true
			}

			delet := f(world.GetParent(id))
			if delet {
				world.Columns.delete.Store(id.Index(), true)
			}
			return delet
		}

		for id := range ecs.All(&world.Parent) {
			f(id)
		}
	}

	// Remove entities that were scheduled for removal
	for index := range world.Columns.delete.Ones() {
		id := world.Table.IDs().Index(index)

		world.DeleteEntityImmediately(id)
	}
	world.Columns.delete.Reset()
}
