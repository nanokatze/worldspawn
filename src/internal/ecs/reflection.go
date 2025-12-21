package ecs

import (
	"iter"
	"reflect"
)

// TODO: rename this pls
type AnyColumn interface {
	Reflect() ReflectedColumn
}

// TODO: move inside a subpackage?
type ReflectedColumn interface {
	ElemType() reflect.Type
	All() iter.Seq[Entity]
	Get(id Entity, v reflect.Value) bool
	Set(id Entity, v reflect.Value)
	Delete(id Entity)
	Copy(src ReflectedColumn)
}
