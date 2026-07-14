package game

import (
	"cmp"
	"slices"

	"worldspawn/internal/ecs"
)

// TODO: rename this file

// I'd probably prefer we keep the vague outline of our ecs.Column stuff except
// it's all interfaces and everything is indexed with an int rather than ecs.ID.
// We would otherwise validate stuff the same. Entity2 will smooth things over
// during programming.

// TODO: if we get rid of reference to the world and replace it with simpler
// structure we could probably factor it out into common code.
// TODO: to allow for parallel enqueue we'll need mutexes too
// TODO: invalidate IO to capture uninentional capture, etc.
type IO struct {
	// TODO: IO doesn't need World, we should replace this with a collection of
	// buffers to enqueue the update funcs into
	world *World

	// TODO: can be anything that's ordered. In fact probably in some cases
	// we'll stuff weird bits into the key and not just a straightforward entity
	// ID.
	key ecs.ID
}

type updatef struct {
	key ecs.ID
	f   func(info *UpdateParams, entity Entity2, io IO)
}

// TODO: shorter names
func (io IO) EnqueueEntityUpdate(to Entity2, f func(info *UpdateParams, entity Entity2, io IO)) {
	updates := &io.world.entityUpdates[to.id.Index()]
	*updates = append(*updates, updatef{io.key, f})
}

func (io IO) EnqueueCreateEntity(f func(info *UpdateParams, entity Entity2, io IO)) {
	io.EnqueueGlobalUpdate(func(info *UpdateParams, world *World) {
		entity := world.CreateEntity(info)
		f(info, entity, IO{world, entity.id})
	})
}

// TODO: kill this and move all globals into Worldspawn? We first would need to
// figure out how to deal with stuff like e.g. rules and stuff being set on
// Worldspawn.
func (io IO) EnqueueGlobalUpdate(f func(info *UpdateParams, world *World)) {
	io.world.globalUpdates = append(io.world.globalUpdates, f)
}

func (world *World) processUpdates(info *UpdateParams) {
	for pass := 0; ; pass++ {
		// TODO: livelock detection

		world.entityUpdates, world.entityUpdates2 = world.entityUpdates2, world.entityUpdates

		progress := false

		for index, updates := range world.entityUpdates2 {
			if len(updates) == 0 {
				continue
			}

			// Sort by key

			slices.SortStableFunc(updates, func(a, b updatef) int { return cmp.Compare(a.key, b.key) })

			// TODO: deterministically permute updates up to key to weed out the bugs

			id := world.Table.IDs().Index(index)

			for _, u := range updates {
				u.f(info, Entity2{world, id}, IO{world, id})
			}

			world.entityUpdates2[index] = nil
		}

		if len(world.globalUpdates) > 0 {
			for _, f := range world.globalUpdates {
				f(info, world)
			}

			world.globalUpdates = nil

			progress = true
		}

		if !progress {
			break
		}
	}
}
