package bitslice

import (
	"testing"
)

func TestFindAndSet(t *testing.T) {
	bs := Make(1)
	bs.Swap(0, true)
	if bs.FindAndSet(0) != -1 {
		t.Errorf("want -1")
	}
}
