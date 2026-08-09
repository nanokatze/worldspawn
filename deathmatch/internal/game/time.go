package game

import (
	"cmp"
	"fmt"
	"math"
	"time"
)

type Time struct {
	// Nanoseconds since the start
	T int64
}

/*
func (t Time) After(u Time) bool { return t.Compare(u) > 0 }

func (t Time) Before(u Time) bool { return t.Compare(u) < 0 }
*/

func (t Time) Compare(u Time) int { return cmp.Compare(t.T, u.T) }

func (t Time) Add(d time.Duration) Time {
	// if t.T == math.MaxInt64 || t.T == math.MinInt64 {
	// 	return t
	// }

	sum := t.T + int64(d)
	if (sum > t.T) != (d > 0) {
		// Saturate instead of overflowing
		if d > 0 {
			return Time{math.MaxInt64}
		} else {
			return Time{math.MinInt64}
		}
	}

	return Time{sum}
}

func (t Time) Sub(u Time) time.Duration {
	return time.Duration(t.T - u.T)
}

func (t Time) String() string {
	// TODO: verify this works correctly with negative t.T and see if we can get
	// rid of fmt.Sprintf
	return fmt.Sprintf("%d.%09ds", t.T/1e9, t.T%1e9)
}
