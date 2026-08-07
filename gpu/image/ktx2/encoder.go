package ktx2

import (
	"encoding/binary"
	"io"

	"worldspawn/gpu"
	"worldspawn/gpu/vk/formatutil"
)

// TODO: change this to a const string
var magic = [...]byte{0xAB, 0x4B, 0x54, 0x58, 0x20, 0x32, 0x30, 0xBB, 0x0D, 0x0A, 0x1A, 0x0A}

func Encode(w io.WriterAt, img *gpu.Image) error {
	config := img.Config()

	var extent [3]int
	copy(extent[:], config.Extent())

	header := fileHeader{
		Magic:    magic,
		Format:   uint32(config.Format()),
		TypeSize: 1, // TODO: fill in the correct value
		Extent: [3]uint32{
			uint32(extent[0]),
			uint32(extent[1]),
			uint32(extent[2]),
		},
		LayerCount: uint32(config.Layers()),
		FaceCount:  1, // TODO: we'll need to do some work to handle cube images correctly
		LevelCount: uint32(config.Mips()),
	}

	levelIndex := make([]levelIndexEntry, config.Mips())

	offset := uint64(80 + 24*config.Mips())
	for i := config.Mips() - 1; i >= 0; i-- {
		// stupidest way ever to ensure alignment lol
		for offset%4 != 0 {
			offset++
		}

		// TODO: this is not correct for block compressed images, we need to round up each side to

		formatDesc := formatutil.Describe(config.Format())

		length := config.Layers() * formatDesc.BlockSize
		for _, side := range config.Extent() {
			length *= max(side>>i, 1)
		}

		levelIndex[i] = levelIndexEntry{
			Offset:             offset,
			Length:             uint64(length),
			UncompressedLength: uint64(length),
		}

		offset += uint64(length)
	}

	ow := io.NewOffsetWriter(w, 0)
	if err := binary.Write(ow, binary.LittleEndian, &header); err != nil {
		return err
	}
	if err := binary.Write(ow, binary.LittleEndian, &levelIndex); err != nil {
		return err
	}

	scratchSize := config.Layers() * int(formatutil.Describe(config.Format()).BlockSize)
	for _, side := range config.Extent() {
		scratchSize *= side
	}

	var g gpu.JobQueue

	tmp := gpu.MakeSliceUncached[byte](scratchSize)
	defer gpu.Free(gpu.UnsafePointer(gpu.SliceData(tmp)))

	for i := config.Mips() - 1; i >= 0; i-- {
		mip := img.SubImage(gpu.SliceMips{i, i + 1})
		gpu.EnqueueCopyImageToMemory(&g,
			tmp, 0, 0,
			mip, nil, mip.Extent())
		enqueueWriteAt(&g, w, tmp.Slice(0, int(levelIndex[i].Length)), int64(levelIndex[i].Offset))
	}

	g.Idle().Wait()

	return nil
}
