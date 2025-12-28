package matc

import (
	"reflect"
	"slices"
	"unsafe"

	"worldspawn/internal/pathtracer/internal/material"
)

// TODO: could we make passing parameters to the material more typesafe?

// TODO: kill in favor of lambdas
type PropertyBag interface {
	// GeometryAttribute(name string) int

	// TODO: this should also provide a way to output strings (for texture
	// descriptors)
	UniformAttribute(name string, out *[4]float32) bool
}

type Preamble func(dst []byte, props PropertyBag)

// TODO: instead of a single PropertyBag we should take separate lambdas and
// stuff tbh.
// TODO: pass ParamsLayout to the preamble rather than during preamble
// compilation?
func CompilePreamble(paramsLayout ParamsLayout, preamble []string) Preamble {
	preamble = slices.Clone(preamble)
	return func(dst []byte, props PropertyBag) {
		dst2 := reflect.NewAt(paramsLayout.structType, unsafe.Pointer(unsafe.SliceData(dst))).Elem()
		for i, f := range preamble {
			var v [4]float32
			if !props.UniformAttribute(f, &v) {
				v = [4]float32{}
			}
			p := dst2.Field(i).Addr().Interface().(*material.AttributeDescriptor)
			*p = material.UniformAttribute(v)
		}
	}
}
