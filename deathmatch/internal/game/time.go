package game

import (
	"cmp"
	"time"
)

type Time struct{ Nanoseconds int64 }

func (t Time) After(u Time) bool { return t.Compare(u) > 0 }

func (t Time) Before(u Time) bool { return t.Compare(u) < 0 }

func (t Time) Compare(u Time) int { return cmp.Compare(t.Nanoseconds, u.Nanoseconds) }

func (t Time) Add(d time.Duration) Time {
	return Time{int64(t.Nanoseconds) + int64(d)}
}

func (t Time) Sub(u Time) time.Duration {
	return time.Duration(int64(t.Nanoseconds) - int64(u.Nanoseconds))
}
