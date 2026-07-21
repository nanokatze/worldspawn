package cache

import "sync"

// TODO: come up with an eviction policy

type Cache[K comparable, V any] struct {
	m sync.Map
	f func(K) *V
}

func (c *Cache[K, V]) Get(key K) *V {
	v, ok := c.m.Load(key)
	if ok {
		return v.(*V)
	}
	v, _ = c.m.LoadOrStore(key, c.f(key)) // TODO: make sure we only have one c.f in flight for each key
	return v.(*V)
}

func New[K comparable, V any](f func(K) *V) *Cache[K, V] {
	return &Cache[K, V]{
		f: f,
	}
}
