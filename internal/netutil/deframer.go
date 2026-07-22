package netutil

import "io"

// TODO: move this into cmd/internal/message?

type Deframer struct {
	r   io.Reader
	n   int64
	eof bool
	err error
}

func NewDeframer(r io.Reader) *Deframer {
	return &Deframer{r: r}
}

func (r *Deframer) Read(b []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	if r.eof {
		return 0, io.EOF
	}
	if r.n == 0 {
		n, err := readVarint(r.r)
		if err != nil {
			r.err = err
			return 0, r.err
		}
		if n == 0 {
			r.eof = true
			return 0, io.EOF
		}
		r.n = int64(n)
	}
	n, err := r.r.Read(b[:min(r.n, int64(len(b)))])
	r.n -= int64(n)
	if err != nil {
		r.err = err
	}
	return n, r.err
}

func (r *Deframer) Next() error {
	if r.err != nil {
		return r.err
	}
	if !r.eof {
		// Discard the remainder of the message
		io.Copy(io.Discard, r)
		if r.err != nil {
			return r.err
		}
	}
	r.eof = false
	return nil
}
