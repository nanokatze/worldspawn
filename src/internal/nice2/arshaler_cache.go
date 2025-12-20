package nice2

import (
	"reflect"
	"sync"
)

type arshalerCache struct {
	m sync.Map
	f Arshalers
}

func makeArshalerCache(f Arshalers) Arshalers {
	cache := new(arshalerCache)
	cache.f = f
	return cache.Get
}

func (cache *arshalerCache) Get(t reflect.Type, _ Arshalers) Arshaler {
	if arshaler, ok := cache.m.Load(t); ok {
		return arshaler.(Arshaler)
	}
	return cache.getSlow(t)
}

func (cache *arshalerCache) getSlow(t reflect.Type) Arshaler {
	arshaler, _ := cache.m.LoadOrStore(t, cache.f(t, cache.Get))
	return arshaler.(Arshaler)
}

var DefaultArshalerCache = makeArshalerCache(DefaultArshalers)
