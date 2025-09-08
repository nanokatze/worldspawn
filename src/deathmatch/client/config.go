package main

import (
	"sync"
	"sync/atomic"
)

type Config struct {
	DisableCosmeticOffset bool
	DontInterpolate       bool
}

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
