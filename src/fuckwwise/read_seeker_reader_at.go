package sfx

import (
	"io"
	"sync"
)

type readSeekerReaderAt struct {
	mu  sync.Mutex
	r   io.ReadSeeker
	off int64
}

var _ io.ReaderAt = (*readSeekerReaderAt)(nil)

func ReaderAtFromReadSeeker(r io.ReadSeeker) io.ReaderAt {
	if readerAt, ok := r.(io.ReaderAt); ok {
		return readerAt
	}
	return &readSeekerReaderAt{r: r}
}

func (r *readSeekerReaderAt) ReadAt(b []byte, off int64) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, err := r.r.Seek(off, io.SeekStart); err != nil {
		return 0, err
	}
	return io.ReadFull(r.r, b)
}
