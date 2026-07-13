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

	WorldGlobals
}

// TODO: do not accept updateParams here but raw Δt and flags. We should
// construct UpdateParams ourselves
func (world *World) Step(updateParams UpdateParams) {
	world.Now = world.Now.Add(updateParams.Δt)

	updateParams.WorldGlobals = world.GetEntity2(1).ScriptState().(WorldGlobals)
	updateParams.Now = world.Now

	world.think(&updateParams)

	world.physicsStep(&updateParams)

	world.handleOutOfBoundsEntities(&updateParams)

	for id, a := range ecs.All(&world.SoundEffectState) {
		soundEffect, _ := world.SoundEffect.Get(id)
		if soundEffect.PlayTime.Add(time.Duration(a.LengthInSamples * 1e9 / 48000)).After(world.Now) {
			continue
		}

		soundEffect.Effect = a.Sound
		soundEffect.Attenuation = a.Attenuation
		soundEffect.PlayTime = updateParams.Now
		world.SoundEffect.Set(id, soundEffect)
	}

	world.deleteMarkedEntities()

	// Clear transient columns
	{
		// TODO: we should create a helper for these
		rcolumns := reflect.ValueOf(&world.Columns).Elem()
		ty := rcolumns.Type()
		for i := range rcolumns.NumField() {
			if ty.Field(i).Tag.Get("worldspawn") != "transient" {
				continue
			}
			rcolumns.Field(i).Addr().Interface().(interface{ Clear() }).Clear()
		}
	}
}

func (world *World) think(updateParams *UpdateParams) {
	// TODO: optimize the pass over thinkers by having a shadow column

	// Update systems which are allowed to be queried from Think

	world.updatePhysicsShadow(updateParams)

	// Run thinkers

	for id, scriptName := range ecs.All(&world.Entity) {
		script := Scripts[reflect.TypeOf(scriptName)]
		if script.Think == nil {
			continue
		}

		// TODO: we'll want a timer wheel of sorts to make this fast
		nextThink, _ := world.NextThink.Get(id)
		if updateParams.Now.Before(nextThink) {
			continue
		}

		script.Think(updateParams, world, Entity2{world, id}, IO{world, id})
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

			if world.delete.Load(id.Index()) {
				return true
			}

			delet := f(world.GetParent(id))
			if delet {
				world.delete.Store(id.Index(), true)
			}
			return delet
		}

		for id := range ecs.All(&world.Parent) {
			f(id)
		}
	}

	// Remove entities that were scheduled for removal
	for index := range world.delete.Ones() {
		id := world.Table.IDs().Index(index)

		world.DeleteEntityImmediately(id)
	}
	world.delete.Reset()
}
