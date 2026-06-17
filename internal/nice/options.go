package nice

import (
	"reflect"
)

// TODO: pool this struct
type options struct {
	budget      int
	getArshaler func(reflect.Type) arshaler
}

var defaultOptions = options{
	budget:      1 << 30,
	getArshaler: getDefaultArshaler,
}

type Options func(opts *options)

func collectOptions(opts ...Options) options {
	if len(opts) == 0 {
		return defaultOptions
	}

	dst := defaultOptions
	JoinOptions(opts...)(&dst)
	return dst
}

func JoinOptions(opts ...Options) Options {
	return func(dst *options) {
		for _, opt := range opts {
			opt(dst)
		}
	}
}

func WithBudget(n int) Options {
	if n <= 0 {
		panic("bad")
	}
	return func(opts *options) { opts.budget = n }
}

func WithArshalers(arshalers Arshalers) Options {
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
