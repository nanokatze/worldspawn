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
	f   func(stx ScriptContext, entity Entity) // TODO: introduce a typedef for these?
}

func (io IO) validate(entity Entity) {
	// TODO: we can validate more stuff tbh
	if !entity.IsValid() {
		panic("bad")
	}
}

// func (io IO) Think(to Entity2, )

func (io IO) Update(to Entity, f func(stx ScriptContext, entity Entity)) {
	io.validate(to)

	updates := &io.world.entityUpdates[to.id.Index()]
	*updates = append(*updates, updatef{io.key, f})
}

func (io IO) Create(f func(stx ScriptContext, entity Entity)) {
	io.world.globalUpdates = append(io.world.globalUpdates, func(info *UpdateParams, world *World) {
		entity := world.CreateEntity(info)
		f(ScriptContext{info, IO{world, uint64(entity.id.Index())}}, entity)
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
				u.f(ScriptContext{info, IO{world, uint64(index)}}, Entity{&world.Columns, id})
			}

			world.entityUpdates2[index] = nil

			progress = true
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
