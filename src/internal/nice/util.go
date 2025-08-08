package nice

func loadOrDefault[M ~map[K]V, K comparable, V any](m M, k K, dv V) V {
	return loadOrElse(m, k, func() V { return dv })
}

func loadOrElse[M ~map[K]V, K comparable, V any](m M, k K, dv func() V) V {
	// BUG: this nil check gets us perf
	if m != nil {
		if v, ok := m[k]; ok {
			return v
		}
	}
	return dv()
}
