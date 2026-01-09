package framing

import (
	"bytes"
	"io"
	"slices"
	"testing"
)

// TODO: throw in a fuzz test as well perhaps.

// TODO: a more thorough test
func TestFraming(t *testing.T) {
	msgs := [][]byte{
		[]byte("A"),
		[]byte("B"),
	}

	buf := new(bytes.Buffer)

	w := NewFramer(buf)
	for _, msg := range msgs {
		w.Write(msg)
		w.Next()
	}

	r := NewDeframer(buf)
	for i, want := range msgs {
		got := make([]byte, len(want))
		if _, err := io.ReadFull(r, got); err != nil {
			t.Errorf("%d: err = %v", i, err)
		} else if !slices.Equal(got, want) {
			t.Errorf("%d: got %q, want %q", i, got, want)
		}
		if err := r.Next(); err != nil {
			t.Errorf("%d: r.Next(): err = %v", i, err)
		}
	}
}
