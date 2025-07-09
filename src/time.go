package worldspawn

import (
	"cmp"
	"time"
)

// TODO: hide that it's int64 internally
// type Time struct { T int64 }
type Time int64

func (t Time) After(u Time) bool { return t.Compare(u) > 0 }

func (t Time) Before(u Time) bool { return t.Compare(u) < 0 }

func (t Time) Compare(u Time) int { return cmp.Compare(t, u) }

func (t Time) Add(d time.Duration) Time {
	return Time(int64(t) + int64(d))
}

func (t Time) Sub(u Time) time.Duration {
	return time.Duration(int64(t) - int64(u))
}
