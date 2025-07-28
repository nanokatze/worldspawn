package worldspawn

import (
	"log"

	"github.com/go-json-experiment/json"

	"worldspawn/ecs"
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

// TODO: rename this to something else, e.g. just Prefab?
type PrefabRef struct {
	Entity   ecs.ID // for prefabs constructed at runtime
	Filename string // for on-disk prefabs
}

// TODO: pass fs explicitly, etc. We'll make Data fs and caches per-World,
// likely
func prefab(filename string) *Components {
	f, err := Data.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// TODO: we'll want a rather very custom deserialization code here

	w := new(Components)
	if err := json.UnmarshalRead(f, w, WorldJSONOptions); err != nil {
		log.Fatalf("prefab: %v", err)
	}
	// TODO: fix up filenames after load
	return w
}

func (w *World) SpawnPrefab(prefabRef PrefabRef) ecs.ID {
	e := w.SpawnEntity()
	w.CopyEntities(e, prefab(prefabRef.Filename))
	return e
}

// TODO: this should not be public
func (dst *Components) CopyEntities(entityID ecs.ID, src *Components) {
	// TODO: rewrite to use reflect

	if v, ok := src.Entity.Load(1); ok {
		dst.Entity.Store(entityID, v)
	}
	if v, ok := src.TranslationRotation.Load(1); ok {
		dst.TranslationRotation.Store(entityID, v)
	}
	if v, ok := src.Scale.Load(1); ok {
		dst.Scale.Store(entityID, v)
	}
	if v, ok := src.Viewmodel.Load(1); ok {
		dst.Viewmodel.Store(entityID, v)
	}
	if v, ok := src.RenderingGeometry.Load(1); ok {
		dst.RenderingGeometry.Store(entityID, v)
	}
	if v, ok := src.CollisionGeometry.Load(1); ok {
		dst.CollisionGeometry.Store(entityID, v)
	}
	if v, ok := src.PhysicsMassOverride.Load(1); ok {
		dst.PhysicsMassOverride.Store(entityID, v)
	}
	if v, ok := src.PhysicsInertiaOverride.Load(1); ok {
		dst.PhysicsInertiaOverride.Store(entityID, v)
	}
	if v, ok := src.GravityFactor.Load(1); ok {
		dst.GravityFactor.Store(entityID, v)
	}
}
