package ktx2

import (
	"encoding/binary"
	"io"
	"slices"

	"worldspawn/gpu"
	"worldspawn/gpu/vk/formatutil"
)

// TODO: introduce encoder type to allow encoding progressively?
//
// The granularity at which we can encode and write things out depends on
// whether supercompression is used, and whether supercompression uses global
// data.
//
// When supercompression global data is used, we can only lay out the file after
// we have compressed all mips (we also need key-value data but that seems to be
// available up-front in all cases?). Regardless of the scheme, with
// supercompression, the granularity is usually something coarse. E.g. entire
// mips, large blocks, etc. Otherwise, there's no granularity requirements
// beyond texel block size.

func Encode(w io.WriterAt, img *gpu.Image) error {
	config := img.Config()

	levelIndex := make([]levelIndexEntry, config.Mips())
	levelData := make([][]byte, config.Mips())
	for i := config.Mips() - 1; i >= 0; i-- {
		var g gpu.JobQueue

		level := img.SubImage(gpu.SliceMips{i, i + 1})

		// TODO: this is incorrect for compressed formats
		uncompressedSize := level.Layers() * int(formatutil.Describe(config.Format()).BlockSize)
		for _, side := range level.Extent() {
			uncompressedSize *= side
		}

		// TODO: when we implement supercompression, Length and
		// UncompressedLength might differ

		// TODO: we read this on host so this should have host caching enabled
		uncompressed := gpu.MakeSliceUncached[byte](uncompressedSize)
		gpu.EnqueueCopyImageToMemory(&g,
			uncompressed, 0, 0,
			level, nil, level.Extent())
		gpu.WaitForIdle(&g)

		data := slices.Clone(uncompressed.Value())

		levelIndex[i] = levelIndexEntry{
			Offset:             ^uint64(0),
			UncompressedLength: uint64(uncompressedSize),
			Length:             uint64(len(data)),
		}
		levelData[i] = data
	}

	ow := io.NewOffsetWriter(w, 0)

	var extent [3]uint32
	for i, side := range config.Extent() {
		extent[i] = uint32(side)
	}
	header := fileHeader{
		Magic:      magic,
		Format:     uint32(config.Format()),
		TypeSize:   1, // TODO: fill in the correct value
		Extent:     extent,
		LayerCount: uint32(config.Layers()),
		FaceCount:  1, // TODO: we'll need to do some work to handle cube images correctly
		LevelCount: uint32(config.Mips()),
	}
	if err := binary.Write(ow, binary.LittleEndian, &header); err != nil {
		return err
	}

	offset := uint64(binary.Size(header) + binary.Size(levelIndex))
	for i := config.Mips() - 1; i >= 0; i-- {
		levelIndex[i].Offset = align(offset, 4)
		offset += uint64(len(levelData[i]))
	}

	if err := binary.Write(ow, binary.LittleEndian, &levelIndex); err != nil {
		return err
	}

	for i := config.Mips() - 1; i >= 0; i-- {
		if _, err := ow.WriteAt(levelData[i], int64(levelIndex[i].Offset)); err != nil {
			return err
		}
	}

	return nil
}

func align[T ~uint64](x, multiple T) T {
	// TODO: don't do this stupidity
	for x%multiple != 0 {
		x++
	}
	return x
}
