package wav

import (
	"bytes"
	"testing"
)

func FuzzHeaderParsing(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		_, err := NewReader(bytes.NewReader(data))
		if err != nil {
			return
		}
	})
}
