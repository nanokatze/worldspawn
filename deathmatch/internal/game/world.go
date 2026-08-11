package game

import (
	"io/fs"
	"log/slog"
	"reflect"
	"unique"

	"worldspawn/internal/ecs"
	"worldspawn/internal/ecs/bitset"
	"worldspawn/internal/loaders/skeleton"
	"worldspawn/internal/physics"
)

// TODO: stick it into World or pass it through UpdateInfo
var Data fs.FS

// TODO: add a type for representing non-mutable lists and stuff?

// TODO: make all of the internals private. We'll need to add infrastructure for
// replication to World first.
type World struct {
	NextID            EntityID
	NextIDSpeculative EntityID // TODO: this could be made private

	// TODO: stop embedding this and make it private
	Columns

	entityUpdates, entityUpdates2 [][]updatef
	globalUpdates                 []func(*UpdateParams, *World)

	logger *slog.Logger
}

func (world *World) Reset(n int) {
	world.Table = ecs.NewTable(n)

	columns := reflect.ValueOf(&world.Columns).Elem()
	for i := range columns.Type().NumField() {
		if !columns.Type().Field(i).IsExported() {
			continue
		}
		col, ok := columns.Field(i).Addr().Interface().(interface{ Init(*ecs.Table) })
		if ok {
			col.Init(world.Table)
		}
	}

	world.Columns.pose = make([]skeleton.Pose, n)

	world.entityUpdates = make([][]updatef, n)
	world.entityUpdates2 = make([][]updatef, n)

	// TODO: we could expose an OptimizeBroadPhase call on physicsSystem I suppose.
	world.Columns.physics = physics.NewSystem(
		int(numCollisionLayers),
		collisionLayerToBroadPhaseLayer[:],
		collisionLayerRules)
	world.Columns.physicsBodyExists.Init(world.Table)

	world.Columns.delete = bitset.Make(n)

	// TODO: the user should pass this
	world.logger = slog.Default()
}

// func (world *World) SetLogger(logger *slog.Logger) { world.logger = logger }

func (world *World) Cap() int { return world.Table.IDs().Cap() }

// TODO: make this private?
// TODO: kill this in favor of IO.Create
func (world *World) CreateEntity(info *UpdateParams) Entity {
	// TODO: don't hardcode index ranges

	if info.Speculating {
		id := world.Table.CreateRowAuto(900, 999, &world.NextIDSpeculative)
		// TODO: parent these newly created entites to 1?
		world.Speculation.Set(id, info.Now)
		return Entity{world: &world.Columns, id: id}
	}

	id := world.Table.CreateRowAuto(1, 899, &world.NextID)
	return Entity{world: &world.Columns, id: id}
}

// This is used by client networking to remove entities.
func (world *World) DeleteEntityImmediately(id EntityID) {
	world.Columns.pose[id.Index()] = skeleton.Pose{}
	if _, ok := world.physicsBodyExists.Get(id); ok {
		world.physics.RemoveBody(physics.BodyID(id.Index()))
	}
	world.Table.DeleteRow(id)
}

// TODO: kill all of these helpers

func (world *Columns) GetParent(id EntityID) EntityID {
	parent, _ := world.Parent.Get(id)
	return parent
}

func (world *Columns) SetParent(id, parent EntityID) {
	if parent != 0 {
		world.Parent.Set(id, parent)
	} else {
		// TODO: our panic behavior should be the same as if we tried to Set
		// this id. I.e. if this id isn't valid, we should crash.
		world.Parent.Delete(id)
	}
}

func (world *Columns) GetSkeleton(id EntityID) *skeleton.Skeleton {
	skellyName, _ := world.Skeleton.Get(id)
	if skellyName == (unique.Handle[string]{}) {
		return nil
	}
	return skeletonCache.Get(skellyName)
}

func (world *World) Entity(id EntityID) Entity {
	if !world.Table.IDs().Exists(id) {
		return Entity{}
	}
	return Entity{world: &world.Columns, id: id}
}
