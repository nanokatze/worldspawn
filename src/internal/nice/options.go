package nice

import (
	"maps"
	"reflect"
)

type options struct {
	budget      int
	getArshaler func(reflect.Type) arshaler
}

var defaultOptions = options{
	budget:      1 << 30,
	getArshaler: getDefaultArshaler,
}

type Option func(opts *options)

func collectOptions(opts ...Option) options {
	if len(opts) == 0 {
		return defaultOptions
	}

	result := defaultOptions
	for _, o := range opts {
		o(&result)
	}
	return result
}

func JoinOptions(opts2 ...Option) Option {
	return func(opts *options) {
		for _, o := range opts2 {
			o(opts)
		}
	}
}

func WithBudget(n int) Option {
	if n <= 0 {
		panic("bad")
	}
	return func(opts *options) { opts.budget = n }
}

// TODO: an option to override getDefaultArshaler as the function to poke for
// "default"?

type Arshalers struct {
	m map[reflect.Type]arshaler
}

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

func JoinArshalers(arshalers ...Arshalers) Arshalers {
	joinedArshalers := Arshalers{m: map[reflect.Type]arshaler{}}
	for _, a := range arshalers {
		maps.Insert(joinedArshalers.m, maps.All(a.m))
	}
	return joinedArshalers
}

func WithArshalers(arshalers Arshalers) Option {
	return func(opts *options) {
		opts.getArshaler = func(t reflect.Type) arshaler {
			arshaler, ok := arshalers.m[t]
			if !ok {
				arshaler = getDefaultArshaler(t)
			}
			return arshaler
		}
	}
}
