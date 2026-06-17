package game

import "reflect"

// TODO: kill Entity interface

type Entity interface{ entity() }

var EntityTypes = []reflect.Type{
	reflect.TypeFor[Animtest](),
	reflect.TypeFor[Gladiator](),
	reflect.TypeFor[InFlightGrenade](),
	reflect.TypeFor[PlayerSpawn](),
	reflect.TypeFor[Player](),
	reflect.TypeFor[SceneGlobals](),
	reflect.TypeFor[Testburger](),
	reflect.TypeFor[WeaponGrenadeLauncher](),
	reflect.TypeFor[WeaponPhysgun](),
	reflect.TypeFor[WeaponSniperRifle](),
}
