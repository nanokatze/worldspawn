package protocol

import "io"

type Framer struct {
	w   io.Writer
	err error
}

func NewFramer(w io.Writer) *Framer {
	return &Framer{w: w}
}

func (w *Framer) Write(b []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	if len(b) == 0 {
		return 0, nil
	}
	if w.err = writeVarint(w.w, uint64(len(b))); w.err != nil {
		return 0, w.err
	}
	var n int
	n, w.err = w.w.Write(b)
	return n, w.err
}

func (w *Framer) Next() error {
	if w.err == nil {
		w.err = writeVarint(w.w, 0)
	}
	return w.err
}
