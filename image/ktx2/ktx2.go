package ktx2

import (
	"encoding/binary"
	"errors"
	"io"
	"math/bits"
)

var errKTX2 = errors.New("not a ktx2 file")

type Section32 struct {
	Off, Len uint32
}

type Section64 struct {
	Off, Len uint64
}

const Magic = "\xab\x4b\x54\x58\x20\x32\x30\xbb\x0d\x0a\x1a\x0a"

type Header struct {
	Magic                      [12]byte
	VkFormat                   uint32
	TypeSize                   uint32
	Width, Height, Depth       uint32
	LayerCount                 uint32
	FaceCount                  uint32
	MipLevelCount              uint32
	SupercompressionScheme     uint32
	DataFormatDescriptor       Section32
	KeyValueData               Section32
	SupercompressionGlobalData Section64
}

type Level struct {
	Section64
	UncompressedLen uint64
}

/*
func AreLevelsInOrderOfIncreasingDetail(levels []Level) bool {
	off := uint64(math.MaxUint64)
	for _, l := range levels {
		if off < l.Off {
			return false
		}
		off = l.Off
	}
	return true
}
*/

type File struct {
	Header
	MipLevels []Level
}

func NewFile(r io.Reader) (*File, error) {
	var header Header
	if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
		return nil, err
	}
	if string(header.Magic[:]) != Magic {
		return nil, errKTX2
	}

	if header.Width <= 0 {
		return nil, errKTX2
	}
	_, err := _type(header.Width, header.Height, header.Depth, header.FaceCount == 6, header.LayerCount > 0)
	if err != nil {
		return nil, errKTX2
	}

	width := header.Width
	height := max(header.Height, 1)
	depth := max(header.Depth, 1)

	// layerCount := max(header.LayerCount, 1)

	if header.FaceCount != 1 && header.FaceCount != 6 {
		return nil, errKTX2
	}

	mipLevelCount := max(header.MipLevelCount, 1)
	if mipLevelCount > CompleteMipChainLength(width, height, depth) {
		return nil, errKTX2
	}
	mipLevels := make([]Level, mipLevelCount)
	if err := binary.Read(r, binary.LittleEndian, mipLevels); err != nil {
		return nil, err
	}

	return &File{Header: header, MipLevels: mipLevels}, nil
}

// TODO: see if we can make this less cancerous through some table magic
func _type(width, height, depth uint32, cube, array bool) (uint32, error) {
	if width <= 0 {
		return 0xffffffff, errKTX2
	}

	var nD int
	switch {
	case depth > 0:
		if height <= 0 {
			return 0xffffffff, errKTX2
		}
		nD = 3
	case height > 0:
		nD = 2
	default:
		nD = 1
	}

	if cube {
		if nD != 2 {
			return 0xffffffff, errKTX2
		}
		if width != height {
			return 0xffffffff, errKTX2
		}

		if array {
			return 6, nil
		} else {
			return 3, nil
		}
	}

	switch nD {
	case 1:
		if array {
			return 4, nil
		} else {
			return 0, nil
		}
	case 2:
		if array {
			return 5, nil
		} else {
			return 1, nil
		}
	case 3:
		if array {
			return 0xffffffff, errKTX2
		}
		return 2, nil
	default:
		panic("unreachable")
	}
}

// TODO: this should be private, probably.
func CompleteMipChainLength(width, height, depth uint32) uint32 {
	return uint32(log2_32(max(width, height, depth))) + 1
}

func log2_32(x uint32) int {
	return 32 - 1 - bits.LeadingZeros32(x)
}
