package proto

import "reflect"

type options struct {
	// userArshalers map[reflect.Type]arshaler
}

func (opts *options) getArshaler(t reflect.Type) arshaler {
	return getDefaultArshaler(t)
}
