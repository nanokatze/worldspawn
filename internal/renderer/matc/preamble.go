package matc

import (
	"reflect"
	"slices"
	"unsafe"

	"worldspawn/internal/renderer/internal/material"
)

type Attributes interface {
	// GeometryAttribute(name string) int

	// TODO: this should also provide a way to output strings (for texture
	// descriptors)
	UniformAttribute(name string, out *[4]float32) bool
}

// TODO: make this opaque?
type Preamble struct {
	f func(dst []byte, attrs Attributes)
}

// TODO: pass ParamStructLayout to the preamble rather than during preamble
// compilation?
func CompilePreamble(params ParamsTuple, preamble []string) Preamble {
	preamble = slices.Clone(preamble)
	return Preamble{
		f: func(dst []byte, attrs Attributes) {
			dst2 := reflect.NewAt(params.typ, unsafe.Pointer(unsafe.SliceData(dst))).Elem()
			for i, f := range preamble {
				var v [4]float32
				if !attrs.UniformAttribute(f, &v) {
					v = [4]float32{}
				}
				p := dst2.Field(i).Addr().Interface().(*material.AttributeDescriptor)
				*p = material.UniformAttribute(v)
			}
		},
	}
}

func (preamble Preamble) Call(dst []byte, attrs Attributes) {
	if preamble.f == nil {
		return
	}
	preamble.f(dst, attrs)
}
