package game

import (
	"reflect"
)

// TODO: kill Entity interface

type Entity interface{ entity() }

var EntityTypes = []reflect.Type{
	reflect.TypeFor[AmmoPickup](),
	reflect.TypeFor[Animtest](),
	reflect.TypeFor[DeleteAfter](),
	reflect.TypeFor[Gladiator](),
	reflect.TypeFor[GrenadeInFlight](),
	reflect.TypeFor[PlayerSpawn](),
	reflect.TypeFor[Player](),
	reflect.TypeFor[RocketInFlight](),
	reflect.TypeFor[Testburger](),
	reflect.TypeFor[TriggerKill](),
	reflect.TypeFor[WeaponGrenadeLauncher](),
	reflect.TypeFor[WeaponPhysgun](),
	reflect.TypeFor[WeaponSniperRifle](),
	reflect.TypeFor[WorldGlobals](),
}
