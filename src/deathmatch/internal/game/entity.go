package game

import (
	"reflect"
)

// TODO: make it a marker interface? We could even generate EntityTypes that
// way.
type Entity interface{ entity() }

var EntityTypes = []reflect.Type{
	reflect.TypeFor[Player](),
	reflect.TypeFor[LaunchedGrenade](),
	reflect.TypeFor[LoopedSound](),
	reflect.TypeFor[SceneGlobals](),
	reflect.TypeFor[Testburger](),
	reflect.TypeFor[WeaponGrenadeLauncher](),
	reflect.TypeFor[WeaponSniperRifle](),
}
