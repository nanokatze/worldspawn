package matc

import (
	"reflect"
	"slices"
	"unsafe"

	"worldspawn/internal/renderer/internal/material"
)

type Preamble struct {
	// TODO: make this cacheable/serializable
	f func(dst []byte, getattr func(name string, out *[4]float32))
}

// TODO: pass ParamStructLayout to the preamble rather than during preamble
// compilation?
// TODO: think about built-in attributes like position, etc?
// TODO: scene attributes?
// TODO: speed this up
// TODO: we could change out to be a AttributeDescriptor interface that accepts
// any, then the user's job becomes simply figuring out the reflect.Value
func CompilePreamble(params ParamsTuple, preamble []string) Preamble {
	preamble = slices.Clone(preamble)
	return Preamble{
		f: func(dst []byte, getattr func(name string, out *[4]float32)) {
			dst2 := reflect.NewAt(params.typ, unsafe.Pointer(unsafe.SliceData(dst))).Elem()
			for i, f := range preamble {
				v := [4]float32{0, 0, 0, 1}
				getattr(f, &v)
				p := dst2.Field(i).Addr().Interface().(*material.AttributeDescriptor)
				*p = material.UniformAttribute(v)
			}
		},
	}
}

// TODO: we could change getattr to return reflect.Value and then we'd decide
// how to pack it ourselves.
func (preamble Preamble) Pack(dst []byte, getattr func(name string, out *[4]float32)) {
	if preamble.f == nil {
		return
	}
	preamble.f(dst, getattr)
}
