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
	All() iter.Seq[ID]
	Get(id ID, v reflect.Value) bool
	Set(id ID, v reflect.Value)
	Delete(id ID)
	Copy(src ReflectedColumn)
}
