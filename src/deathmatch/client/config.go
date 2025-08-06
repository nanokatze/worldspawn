package main

import (
	"sync"
	"sync/atomic"
)

// TODO: replace with map[string]any + bunch of helper methods for conversions?
// TODO: our input needs to be action sets like in steam input, how do we store
// that?

type Config struct {
	DisableCosmeticOffset bool
	DontInterpolate       bool
}

/*
type Config2 map[string]any

func confGet[T any](conf Config2) (T, bool) {
	k := reflect.TypeFor[T]().String()
	if v, ok := conf[k]; ok {
		if v, ok := v.(T); ok {
			return v, ok
		}
	}
	return *new(T), false
}
*/

var (
	configMu sync.Mutex // only locked when modifying
	config   atomic.Pointer[Config]
)

func updateConfig(f func(p *Config)) {
	configMu.Lock()
	defer configMu.Unlock()

	tmp := *config.Load()
	f(&tmp)
	config.Store(&tmp)
}
