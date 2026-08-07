package ktx2

import "slices"

// Like slices.Index, but returns len(s) instead of -1.
func index[S ~[]E, E comparable](s S, v E) int {
	if i := slices.Index(s, v); i >= 0 {
		return i
	}
	return len(s)
}
