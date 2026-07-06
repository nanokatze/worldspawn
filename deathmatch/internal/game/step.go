package game

import (
	"log/slog"
	"reflect"
	"time"

	"worldspawn/internal/ecs"
	"worldspawn/internal/physics"
)

// TODO: rename to StepContext or something
type UpdateParams struct {
	// Now         Time // TODO: fill this out in world.Step for now
	Δt          time.Duration
	Speculating bool
	Logger      *slog.Logger
}

func (world *World) Step(updateParams *UpdateParams) {
	world.Now = world.Now.Add(updateParams.Δt)

	world.think(updateParams)

	world.physicsStep(updateParams)

	world.handleOutOfBoundsEntities(updateParams)

	for id, a := range ecs.All(&world.SoundEffectState) {
		soundEffect, _ := world.SoundEffect.Get(id)
		if soundEffect.PlayTime.Add(time.Duration(a.LengthInSamples * 1e9 / 48000)).After(world.Now) {
			continue
		}

		soundEffect.Effect = a.Sound
		soundEffect.Attenuation = a.Attenuation
		soundEffect.PlayTime = world.Now
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
		if world.Now.Before(nextThink) {
			continue
		}

		script.Think(updateParams, world, id, IO{world, id})
	}

	// Process the enqueued updates

	world.processUpdates(updateParams)
}

// TODO: rename to make it clear that we're deleting things already marked for
// deletion.
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

			if _, delet := world.Delete.Get(id); delet {
				return true
			}

			delet := f(world.GetParent(id))
			if delet {
				world.Delete.Set(id, struct{}{})
			}
			return delet
		}

		for id := range ecs.All(&world.Parent) {
			f(id)
		}
	}

	// Remove entities that were scheduled for removal
	for id := range ecs.All(&world.Delete) {
		if _, ok := world.physicsBodyExists.Get(id); ok {
			world.physics.RemoveBody(physics.BodyID(id.Index()))
		}
		world.Table.DeleteRow(id)
	}
}
