package game

import (
	"cmp"
	"slices"
)

// TODO: rename this file to commands.go, IO to Commands and io to cmds

// TODO: if we get rid of reference to the world and replace it with simpler
// structure we could probably factor it out into common code.
// TODO: to allow for parallel enqueue we'll need mutexes too
// TODO: invalidate IO to capture uninentional capture, etc?
type IO struct {
	// TODO: IO doesn't need World, we should replace this with a collection of
	// buffers to enqueue the update funcs into
	world *World

	// Use uint64 so that when processing contact pairs, we can pack indices of
	// both entities in a contact pair.
	key uint64
}

type updatef struct {
	key uint64
	f   func(info *UpdateParams, entity Entity2, io IO)
}

func (io IO) validate(entity Entity2) {
	// TODO: we can validate more stuff tbh
	if !entity.Valid() {
		panic("bad")
	}
}

func (io IO) Update(to Entity2, f func(info *UpdateParams, entity Entity2, io IO)) {
	io.validate(to)

	updates := &io.world.entityUpdates[to.id.Index()]
	*updates = append(*updates, updatef{io.key, f})
}

func (io IO) Create(f func(info *UpdateParams, entity Entity2, io IO)) {
	io.world.globalUpdates = append(io.world.globalUpdates, func(info *UpdateParams, world *World) {
		entity := world.CreateEntity(info)
		f(info, entity, IO{world, uint64(entity.id.Index())})
	})
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
				u.f(info, Entity2{world, id}, IO{world, uint64(index)})
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
