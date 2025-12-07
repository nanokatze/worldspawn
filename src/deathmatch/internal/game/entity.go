package game

import (
	"reflect"
)

type Entity any

// TODO: make it a map?
var EntityTypes = []reflect.Type{
	reflect.TypeFor[FPSCharacter](),
	reflect.TypeFor[GrenadeLauncherGrenade](),
	reflect.TypeFor[LoopedSound](),
	reflect.TypeFor[Weapon2GenericProjectileLauncher](),
	reflect.TypeFor[WeaponGenericMelee](),
	reflect.TypeFor[WeaponGenericProjectileLauncher](),
	reflect.TypeFor[WeaponPhysgun](),
	reflect.TypeFor[WeaponSniperRifle](),
}
