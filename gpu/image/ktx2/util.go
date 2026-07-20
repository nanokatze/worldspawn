package ktx2

// Like slices.Index, but returns len(s) instead of -1.
func index[S ~[]E, E comparable](s S, e E) int {
	i := 0
	for ; i < len(s); i++ {
		if s[i] == e {
			break
		}
	}
	return i
}
