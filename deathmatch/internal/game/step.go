package game

import (
	"log/slog"
	"time"

	"worldspawn/internal/ecs"
)

// TODO: rename to StepContext or something
type UpdateParams struct {
	Δt          time.Duration
	Speculating bool

	Logger *slog.Logger

	Worldspawn
}

// TODO: pass Δt and flags explicitly and then make UpdateParams private
// TODO: make this a standalone function too
func Step(world *World, updateParams UpdateParams) {
	{
		worldspawn := world.Entity(1)

		state := worldspawn.ScriptState().(Worldspawn)
		state.Now = state.Now.Add(updateParams.Δt)
		worldspawn.SetScriptState(state)

		updateParams.Worldspawn = state
	}

	processUpdates(world, &updateParams)

	// Update the shadows for thinkers
	//
	// TODO: not sure if we should do it here or if the passes should do it
	// themselves. There's definitely arguments for either.
	updateShadows(world, &updateParams)

	think(world, &updateParams)
	processUpdates(world, &updateParams)

	physicsStep(world, &updateParams)

	markEntitiesOutOfBounds(world, &updateParams)

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

	deleteMarkedEntities(world)
}

func updateShadows(world *World, updateParams *UpdateParams) {
	updatePhysicsShadow(world, updateParams)
}

// TODO: make it public so that client replication code can use this? Client
// replication code still needs World.DeleteEntityImmediately. I guess it could
// also do whatever surgery it needs.
func deleteMarkedEntities(world *World) {
	// Propagate deletion from parents.
	//
	// TODO: make this less gross. We could do a probe whether there's any
	// deletions at all right now.
	{
		var f func(id EntityID) bool
		// TODO: we could also rotate this
		f = func(id EntityID) bool {
			if id == 0 {
				return false
			}

			if world.Entities.delete.Load(id.Index()) {
				return true
			}

			delet := f(world.GetParent(id))
			if delet {
				world.Entities.delete.Store(id.Index(), true)
			}
			return delet
		}

		for id := range ecs.All(&world.Parent) {
			f(id)
		}
	}

	// Remove entities that were scheduled for removal
	for index := range world.Entities.delete.Ones() {
		id := world.Table.IDs().Index(index)

		world.DeleteEntityImmediately(id)
	}
	world.Entities.delete.Reset()
}
