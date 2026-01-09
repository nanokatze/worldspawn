package nice

import (
	"maps"
	"reflect"
)

// TODO: an option to override getDefaultArshaler as the function to poke for
// "default"?

type Arshalers struct {
	m map[reflect.Type]arshaler
}

// TODO: allow nils
func MakeArshaler[T any](marshal func(enc *Encoder, v *T) error, unmarshal func(dec *Decoder, v *T) error) Arshalers {
	t := reflect.TypeFor[T]()
	return Arshalers{
		m: map[reflect.Type]arshaler{
			t: {
				marshal: func(enc *Encoder, v reflect.Value) error {
					return marshal(enc, v.Addr().Interface().(*T))
				},
				unmarshal: func(dec *Decoder, v reflect.Value) error {
					return unmarshal(dec, v.Addr().Interface().(*T))
				},
			},
		},
	}
}

func JoinArshalers(arshalerss ...Arshalers) Arshalers {
	dst := Arshalers{m: map[reflect.Type]arshaler{}}
	for _, a := range arshalerss {
		maps.Insert(dst.m, maps.All(a.m))
	}
	return dst
}
