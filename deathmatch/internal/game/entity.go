package game

import "reflect"

// TODO: this is actually "object logic"

type Entity interface{ entity() }

var EntityTypes = []reflect.Type{
	reflect.TypeFor[Character](),
	reflect.TypeFor[DeleteAfter](),
	reflect.TypeFor[DroppedWeapon](),
	reflect.TypeFor[LaunchedGrenade](),
	reflect.TypeFor[PlayerSpawn](),
	reflect.TypeFor[Player](),
	reflect.TypeFor[SceneGlobals](),
	reflect.TypeFor[Testburger](),
	reflect.TypeFor[WeaponGrenadeLauncher](),
	reflect.TypeFor[WeaponSniperRifle](),
}
