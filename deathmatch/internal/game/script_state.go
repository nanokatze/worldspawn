package game

import "reflect"

// TODO: rename to ScriptState
type Entity interface{}

// TODO: kill this
var EntityTypes = []reflect.Type{
	reflect.TypeFor[AmmoPickup](),
	reflect.TypeFor[Animtest](),
	reflect.TypeFor[DeleteAfter](),
	reflect.TypeFor[ExplosiveBarrel](),
	reflect.TypeFor[Gladiator](),
	reflect.TypeFor[GrenadeInFlight](),
	reflect.TypeFor[PlayerSpawn](),
	reflect.TypeFor[Player](),
	reflect.TypeFor[RocketInFlight](),
	reflect.TypeFor[Testburger](),
	reflect.TypeFor[TriggerKill](),
	reflect.TypeFor[WeaponAssaultRifle](),
	reflect.TypeFor[WeaponGrenadeLauncher](),
	reflect.TypeFor[WeaponPhysgun](),
	reflect.TypeFor[Worldspawn](),
}
