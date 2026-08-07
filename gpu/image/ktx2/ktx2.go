package ktx2 // TODO: implement common handling in gpu/image like Go's core image.Decode does?

import (
	"io"

	"worldspawn/gpu"
)

type fileHeader struct {
	Magic                      [12]byte
	Format                     uint32
	TypeSize                   uint32
	Extent                     [3]uint32
	LayerCount                 uint32
	FaceCount                  uint32
	LevelCount                 uint32
	SupercompressionScheme     uint32
	DataFormatDescriptor       indexEntry32
	KeyValueData               indexEntry32
	SupercompressionGlobalData indexEntry64
}

type indexEntry32 struct {
	Offset uint32
	Length uint32
}

type indexEntry64 struct {
	Offset uint64
	Length uint64
}

type levelIndexEntry struct {
	indexEntry64
	UncompressedLength uint64
}

// TODO: move this into gpu?
func enqueueReadAt(jq *gpu.JobQueue, r io.ReaderAt, p gpu.Slice[byte], off int64) {
	enqueueHostCall(jq, func() {
		if _, err := r.ReadAt(p.Value(), off); err != nil {
			// We don't really have any way to report read failures for now.
			panic(err)
		}
	})
}

func enqueueWriteAt(jq *gpu.JobQueue, w io.WriterAt, p gpu.Slice[byte], off int64) {
	enqueueHostCall(jq, func() {
		if _, err := w.WriteAt(p.Value(), off); err != nil {
			// We don't really have any way to report read failures for now.
			panic(err)
		}
	})
}

// TODO: move into gpu?
// TODO: come up with a better name?
func enqueueHostCall(jq *gpu.JobQueue, f func()) {
	var wg1 gpu.WaitGroup
	wg1.Add(1)
	var wg2 gpu.WaitGroup
	wg2.Add(1)

	wg1.EnqueueDone(jq)
	go func() {
		wg1.Wait()
		f()
		wg2.Done()
	}()
	wg2.EnqueueWait(jq)
}
