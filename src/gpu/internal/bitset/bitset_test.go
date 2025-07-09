package bitset

import (
	"testing"
)

func TestFindAndSet(t *testing.T) {
	bs := Make(1)
	bs.Set(0)
	if bs.FindAndSet(0) != -1 {
		t.Errorf("want -1")
	}
}
