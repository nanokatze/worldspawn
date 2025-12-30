package game

import (
	"reflect"
)

type Entity any

var EntityTypes = []reflect.Type{
	reflect.TypeFor[FPSCharacter](),
	reflect.TypeFor[GrenadeLauncherGrenade](),
	reflect.TypeFor[LoopedSound](),
	reflect.TypeFor[SceneProperties](),
	reflect.TypeFor[Testburger](),
	reflect.TypeFor[WeaponGenericProjectileLauncher](),
}
