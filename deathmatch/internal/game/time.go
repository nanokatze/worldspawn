package game

import (
	"cmp"
	"fmt"
	"time"
)

type Time struct {
	// Nanoseconds since the start
	T int64
}

func (t Time) After(u Time) bool { return t.Compare(u) > 0 }

func (t Time) Before(u Time) bool { return t.Compare(u) < 0 }

func (t Time) Compare(u Time) int { return cmp.Compare(t.T, u.T) }

func (t Time) Add(d time.Duration) Time { return Time{t.T + int64(d)} }

func (t Time) Sub(u Time) time.Duration { return time.Duration(t.T - u.T) }

func (t Time) String() string {
	// TODO: verify this works correctly with negative t.T and see if we can get
	// rid of fmt.Sprintf
	return fmt.Sprintf("%d.%09ds", t.T/1e9, t.T%1e9)
}
