package game

import (
	"reflect"
)

type Entity any

var EntityTypes = []reflect.Type{
	reflect.TypeFor[FPSCharacter](),
	reflect.TypeFor[GrenadeLauncherGrenade](),
	reflect.TypeFor[LoopedSound](),
	reflect.TypeFor[SceneGlobals](),
	reflect.TypeFor[Testburger](),
	reflect.TypeFor[WeaponGrenadeLauncher](),
}
