package matc

import (
	"reflect"
	"strconv"

	"worldspawn/internal/compiler"
	"worldspawn/internal/pathtracer/internal/material"
)

type ParamsTuple struct {
	typ reflect.Type // TODO: rename so that it's clear that it's refle4ct.Type
}

func MakeParamsTuple(paramTypes []compiler.Type) ParamsTuple {
	fields := make([]reflect.StructField, len(paramTypes))
	for i, t := range paramTypes {
		fields[i] = reflect.StructField{
			Name: "A" + strconv.Itoa(i),
			Type: asdasd(t),
		}
	}
	return ParamsTuple{typ: reflect.StructOf(fields)}
}

// TODO: naming
func asdasd(t compiler.Type) reflect.Type {
	switch t.(type) {
	case AttributeDescriptorType:
		return reflect.TypeFor[material.AttributeDescriptor]()
	}

	panic("bad")
}

func (layout ParamsTuple) Num() int {
	if layout.typ == nil {
		return 0
	}
	return layout.typ.NumField()
}

// TODO: make these use int as well?
func (layout ParamsTuple) Size() int {
	if layout.typ == nil {
		return 0
	}
	return int(layout.typ.Size())
}

func (layout ParamsTuple) Offset(i int) int {
	return int(layout.typ.Field(int(i)).Offset)
}
