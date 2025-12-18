package nice2

import (
	"fmt"
	"reflect"
	"sync"
)

type ArshalerMap struct{ m sync.Map }

func WithArshaler(t reflect.Type, makeArshaler func(ArshalerGetter) Arshaler) func(m *ArshalerMap) {
	return func(m *ArshalerMap) {
		_, ok := m.m.LoadOrStore(t, makeArshaler(m.Get))
		if ok {
			panic(fmt.Sprintf("already have marshaler defined for type %v", t))
		}
	}
}

func WithTypedArshaler[T any](makeTypedArshaler func(ArshalerGetter) TypedArshaler[T]) func(m *ArshalerMap) {
	makeArshaler := func(getArshaler ArshalerGetter) Arshaler {
		return makeTypedArshaler(getArshaler).Arshaler()
	}
	return WithArshaler(reflect.TypeFor[T](), makeArshaler)
}

func MakeArshalerMap(opts ...func(m *ArshalerMap)) *ArshalerMap {
	var m ArshalerMap
	for _, o := range opts {
		o(&m)
	}
	return &m
}

func (m *ArshalerMap) Get(t reflect.Type) Arshaler {
	if arshaler, ok := m.m.Load(t); ok {
		return arshaler.(Arshaler)
	}
	arshaler, _ := m.m.LoadOrStore(t, defaultArshalerGetter(m.Get)(t))
	return arshaler.(Arshaler)
}
