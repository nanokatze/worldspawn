package netutil

import (
	"bytes"
	"io"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	frames := [][][]byte{
		{
			[]byte("A"),
			[]byte("B"),
			[]byte("C"),
		},
		{
			[]byte("1"),
			[]byte("2"),
			[]byte("3"),
		},
	}

	buf := new(bytes.Buffer)

	framer := NewFramer(buf)
	for _, frame := range frames {
		for _, chunk := range frame {
			framer.Write(chunk)
		}
		framer.Next()
	}

	deframer := NewDeframer(buf)
	for frameIdx, frame := range frames {
		for chunkIdx, wantChunk := range frame {
			chunk := make([]byte, len(wantChunk))
			if _, err := io.ReadFull(deframer, chunk); err != nil {
				t.Fatalf("frames[%d][%d]: err = %v, want nil", frameIdx, chunkIdx, err)
			}
			if !bytes.Equal(chunk, wantChunk) {
				t.Fatalf("frames[%d][%d]: chunk = %q, want %q", frameIdx, chunkIdx, chunk, wantChunk)
			}
		}
		if _, err := deframer.Read(make([]byte, 1)); err != io.EOF {
			t.Fatalf("frames[%d]: err = %v, want %v", frameIdx, err, io.EOF)
		}
		if err := deframer.Next(); err != nil {
			t.Fatalf("frames[%d]: err = %v, want nil", frameIdx, err)
		}
	}
}

func FuzzDeframer(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		deframer := NewDeframer(bytes.NewReader(data))
		for {
			io.ReadAll(deframer)
			if deframer.Next() != nil {
				break
			}
		}
	})
}
