package game

import (
	"reflect"
	"time"

	"worldspawn/internal/ecs"
)

func (world *World) Step(updateParams *UpdateParams) {
	world.Now = world.Now.Add(updateParams.Δt)

	// TODO: optimize loops over entities implementing particular interface by
	// having shadow columns.

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

	// TODO: update physics shadow here so that the physics world doesn't
	// include deleted entities.

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
