package matc

import (
	"math"
	"reflect"
	"strconv"

	"worldspawn/gpu"
	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
)

// TODO: rename this file pls

func asdasd(t compiler.Type) reflect.Type {
	switch t := t.(type) {
	case core.IntType:
		switch t.N {
		case 32:
			return reflect.TypeFor[int32]()
		}

	case AttributeDescriptor:
		return reflect.TypeFor[gpu.UnsafePointer]()
	}

	panic("bad")
}

func ParamStruct(paramTypes []compiler.Type) reflect.Type {
	fields := make([]reflect.StructField, len(paramTypes))
	for i, t := range paramTypes {
		fields[i] = reflect.StructField{
			Name: "A" + strconv.Itoa(i),
			Type: asdasd(t),
		}
	}

	return reflect.StructOf(fields)
}

// TODO: aaaaaaa grrrrr????
// TODO: beside src this would also need renderer.Mesh (which we need to move
// out of here) to pack attributes.
// TODO: dst should really just be a byte bag tbh. Or idk.
func GatherArgs(dst, src reflect.Value, m []string) {
	for i, f := range m {
		v := src.FieldByName(f)
		if v.Type().Kind() == reflect.Float32 {
			v = reflect.ValueOf(int32(math.Float32bits(float32(v.Float()))))
		}
		dst.Field(i).Set(v)
	}
}
