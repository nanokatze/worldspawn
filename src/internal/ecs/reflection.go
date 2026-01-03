package ecs

import (
	"iter"
	"reflect"
)

// TODO: eventually rename Column[T] struct to something else and rename this
// interface to Column?
type AnyColumn interface {
	Reflect() ReflectedColumn // TODO: kill the reflected column interface?
}

type ReflectedColumn interface {
	ElemType() reflect.Type
	All() iter.Seq[ID]
	Get(id ID, vOut reflect.Value) bool
	Set(id ID, v reflect.Value)
	Delete(id ID)
	Copy(src ReflectedColumn)
}
