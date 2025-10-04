package nice

import "reflect"

const defaultBudget = 1 << 30

type options struct {
	customMarshalers   map[reflect.Type]marshaler
	customUnmarshalers map[reflect.Type]unmarshaler
	budget             int
}

type Option func(opts *options)

func collectOptions(opts ...Option) options {
	// ughhhhhhhhh
	if len(opts) == 0 {
		return options{budget: defaultBudget}
	}

	// TODO: this is kinda slow with small marshals and unmarshals because
	// result leaks and causes an allocation. We can optimize certain happy
	// cases by replacing Option with a private interface and handling certain
	// cases in a special way.

	var result options
	result.budget = defaultBudget
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

func WithMarshaler[T any](fn func(enc *Encoder, v *T) error) Option {
	return func(opts *options) {
		// TODO: rename
		fn2 := func(enc *Encoder, v reflect.Value) error {
			return fn(enc, v.Addr().Interface().(*T))
		}

		if opts.customMarshalers == nil {
			opts.customMarshalers = make(map[reflect.Type]marshaler)
		}
		opts.customMarshalers[reflect.TypeFor[T]()] = fn2
	}
}

func WithUnmarshaler[T any](fn func(dec *Decoder, v *T) error) Option {
	return func(opts *options) {
		// TODO: rename
		fn2 := func(dec *Decoder, v reflect.Value) error {
			return fn(dec, v.Addr().Interface().(*T))
		}

		if opts.customUnmarshalers == nil {
			opts.customUnmarshalers = make(map[reflect.Type]unmarshaler)
		}
		opts.customUnmarshalers[reflect.TypeFor[T]()] = fn2
	}
}

func WithBudget(n int) Option {
	if n <= 0 {
		panic("bad")
	}
	return func(opts *options) { opts.budget = n }
}
