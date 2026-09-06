package ktx2

import (
	"encoding/binary"
	"io"
	"slices"

	"worldspawn/gpu"
)

// TODO: supercompression

type encoderOptions struct{}

// TODO: use io.Writer instead. WriterAt would only be warranted with a
// progressive encoder.
func Encode(w io.WriterAt, img *gpu.Image, opts ...func(*encoderOptions)) error {
	config := img.Config()

	mipHeaders := make([]mipSectionHeader, config.Mips())
	mipData := make([][]byte, config.Mips())
	for i := range config.Mips() {
		var g gpu.JobQueue

		img_i := img.SubImage(gpu.SliceMips{i, i + 1})

		linearSize := calcLinearSize(config.Format(), config.Extent(), config.Layers())

		// TODO: we read this on host so this should have host caching enabled
		linear := gpu.MakeSliceUncached[byte](linearSize)
		gpu.EnqueueCopyImageToMemory(&g,
			linear, 0, 0,
			img_i, nil, img_i.Extent())
		gpu.WaitForIdle(&g)

		data := slices.Clone(linear.Value())

		mipHeaders[i] = mipSectionHeader{
			Offset:             ^uint64(0),
			Length:             uint64(len(data)),
			UncompressedLength: uint64(linearSize),
		}
		mipData[i] = data
	}

	ow := io.NewOffsetWriter(w, 0)

	var header fileHeader
	header.Magic = magic
	header.Format = uint32(config.Format())
	header.TypeSize = 1 // TODO: fill in the correct value
	for i, side := range config.Extent() {
		header.Extent[i] = uint32(side)
	}
	header.LayerCount = uint32(config.Layers())
	header.FaceCount = 1 // TODO: we'll need to do some work to handle cube images correctly
	header.MipCount = uint32(config.Mips())
	if err := binary.Write(ow, binary.LittleEndian, &header); err != nil {
		return err
	}

	offset := uint64(binary.Size(header) + binary.Size(mipHeaders))
	for i := config.Mips() - 1; i >= 0; i-- {
		mipHeaders[i].Offset = roundUp(offset, 4)
		offset += uint64(len(mipData[i]))
	}

	if err := binary.Write(ow, binary.LittleEndian, &mipHeaders); err != nil {
		return err
	}

	for i := config.Mips() - 1; i >= 0; i-- {
		if _, err := ow.WriteAt(mipData[i], int64(mipHeaders[i].Offset)); err != nil {
			return err
		}
	}

	return nil
}
