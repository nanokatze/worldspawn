package matc

import (
	"reflect"
	"strconv"

	"worldspawn/internal/compiler"
	"worldspawn/internal/pathtracer/internal/material"
)

// TODO: should this be called paramLayout?
type ParamsLayout struct {
	structType reflect.Type
}

func LayoutParams(paramTypes []compiler.Type) ParamsLayout {
	fields := make([]reflect.StructField, len(paramTypes))
	for i, t := range paramTypes {
		fields[i] = reflect.StructField{
			Name: "A" + strconv.Itoa(i),
			Type: asdasd(t),
		}
	}
	return ParamsLayout{structType: reflect.StructOf(fields)}
}

// TODO: naming
func asdasd(t compiler.Type) reflect.Type {
	switch t.(type) {
	case AttributeDescriptorType:
		return reflect.TypeFor[material.AttributeDescriptor]()
	}

	panic("bad")
}

func (layout ParamsLayout) Num() int {
	if layout.structType == nil {
		return 0
	}
	return layout.structType.NumField()
}

// TODO: make these use int as well?
func (layout ParamsLayout) Size() int {
	if layout.structType == nil {
		return 0
	}
	return int(layout.structType.Size())
}

func (layout ParamsLayout) Offset(i int) int {
	return int(layout.structType.Field(int(i)).Offset)
}
