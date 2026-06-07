package game

import (
	"log"
	"path/filepath"

	"github.com/go-json-experiment/json"

	"worldspawn/internal/ecs"
)

/*
Prefabs

Prefabs represent a template of one or more entities that can be instantinated.
We use prefabs in two ways:

1) referencing premade structures in maps so that when a prefab is changed, that
   change is reflected in the maps referencing the prefab, and

2) specifying the entity to spawn as seen in projectile weapons

We might want to be able to create and destroy prefabs at runtime to cover use
cases such as created-at-runtime weapons and projectiles, but also avoid sending
unnecessary traffic over the network.

During iteration, we also want a way to re-instantinate already instantinated
prefabs to reflect the changes in a prefab.

The current solution

Prefabs created at runtime are represented as entities with a Prefab component.
These entities can be children of other entities, so that a prefab is
automatically removed when the parent is removed.

Entities refer to prefabs using prefab references. A prefab reference is either
an EntityID pointing to a runtime-created prefab, or a filename pointing to an
on-disk prefab.

Reinstantinating prefabs is currently not implemented, but any solution will
consist of endowing the spawned entities with a component that lets us determine
which prefab was used to spawn the said entity. Then, when reinstantinating, we
can remove the entities that were spawned out of the changed prefab.

Future:

Because we're specifying prefabs as JSON, we might solve the 1st case by
upgrading to a tiny JS engine which will be loading JSON file, manipulating it
and returning a new object.
*/

type CollectionInstance struct {
	Filename string
}

// TODO: rename this to something else, e.g. just Prefab?
// TODO: make it this an interface of file-backed prefab and inline string
// prefab (i.e. just json)
type PrefabRef struct {
	Filename string // for on-disk prefabs
}

// TODO: rename this to make it clear that we're instanting collections
// specified by CollectionInstance components. E.g.
// Realize{,Collection,Prefab}Instances?
func (world *World) InstantinateCollections() {
	for id, collection := range ecs.All(&world.CollectionInstance) {
		world.CollectionInstance.Delete(id)
		world.InstanceCollectionAt(id, PrefabRef{Filename: collection.Filename})
	}
}

// TODO: pass fs explicitly, etc. We'll make Data fs and caches per-World,
// likely
func prefab(filename string) *Columns {
	// TODO: we shouldn't need to be doing filepath.Clean here, the exporter should export stuff properly by itself
	f, err := Data.Open(filepath.Clean(filename))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// TODO: we'll want a rather very custom deserialization code here

	w := new(Columns)
	if err := json.UnmarshalRead(f, w, JSONOptions); err != nil {
		log.Fatalf("prefab: %v", err)
	}
	// TODO: fix up filenames after load
	return w
}

func (world *World) SpawnPrefab(filename string, info *UpdateParams) ecs.ID {
	e := world.CreateEntity(info)
	world.CopyEntities(e, prefab(filename))
	return e
}

// TODO: this should create the collection entities
func (world *World) InstanceCollectionAt(id ecs.ID, prefabRef PrefabRef) {
	T := world.GetTransform(id)

	world.CopyEntities(id, prefab(prefabRef.Filename))
	// TODO: actually compose these rather than override!
	world.SetTransform(id, T)

	// TODO: we also need to take velocity into account
}

// TODO: reorganize collection instantination and remove this
func (dst *Columns) CopyEntities(id ecs.ID, src *Columns) {
	// TODO: rewrite using reflect

	if v, ok := src.Entity.Get(1); ok {
		dst.Entity.Set(id, v)
	}
	if v, ok := src.Script.Get(1); ok {
		dst.Script.Set(id, v)
	}
	if v, ok := src.TransformTR.Get(1); ok {
		dst.TransformTR.Set(id, v)
	}
	if v, ok := src.TransformS.Get(1); ok {
		dst.TransformS.Set(id, v)
	}
	if v, ok := src.RenderingGeometry.Get(1); ok {
		dst.RenderingGeometry.Set(id, v)
	}
	if v, ok := src.CollisionGeometry.Get(1); ok {
		dst.CollisionGeometry.Set(id, v)
	}
	if v, ok := src.PhysicsMassOverride.Get(1); ok {
		dst.PhysicsMassOverride.Set(id, v)
	}
	if v, ok := src.PhysicsInertiaOverride.Get(1); ok {
		dst.PhysicsInertiaOverride.Set(id, v)
	}
	if v, ok := src.GravityFactor.Get(1); ok {
		dst.GravityFactor.Set(id, v)
	}
}
