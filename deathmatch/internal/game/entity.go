package game

import "reflect"

// In preparation for scripts, replace Entity interface with a map of map of
// functions for now I suppose.

type Entity interface{ entity() }

var EntityTypes = []reflect.Type{
	reflect.TypeFor[Animtest](),
	reflect.TypeFor[Gladiator](),
	reflect.TypeFor[PlayerSpawn](),
	reflect.TypeFor[Player](),
	reflect.TypeFor[SceneGlobals](),
	reflect.TypeFor[Testburger](),
	reflect.TypeFor[WeaponGrenadeLauncher](),
	reflect.TypeFor[WeaponPhysgun](),
	reflect.TypeFor[WeaponSniperRifle](),
}
