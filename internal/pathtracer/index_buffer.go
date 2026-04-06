package pathtracer

import (
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

type IndexType uint8

const (
	IndexNone IndexType = iota
	Index8
	Index16
	Index32
)

func (indexType IndexType) Size() int {
	if indexType == IndexNone {
		return 0
	}
	return 1 << (int(indexType) - 1)
}

// TODO: make public? Rename to ToVkIndexType()?
func (indexType IndexType) vkIndexType() vk.IndexType {
	switch indexType {
	case IndexNone:
		return vk.INDEX_TYPE_NONE_KHR
	case Index8:
		return vk.INDEX_TYPE_UINT8
	case Index16:
		return vk.INDEX_TYPE_UINT16
	case Index32:
		return vk.INDEX_TYPE_UINT32
	default:
		panic("unreachable")
	}
}

type IndexBuffer struct {
	type_ IndexType
	data  gpu.UnsafePointer
	len   int
}

func MakeIndexBuffer(indexType IndexType, len int) IndexBuffer {
	return IndexBuffer{
		type_: indexType,
		data:  gpu.UnsafePointer(gpu.SliceData(gpu.MakeSliceUncached[byte](len * indexType.Size()))),
		len:   len,
	}
}

func (buf IndexBuffer) Type() IndexType { return buf.type_ }

func (buf IndexBuffer) Len() int { return buf.len }

func (buf IndexBuffer) Slice(i, j int) IndexBuffer {
	if !(0 <= i && i <= j && j <= buf.len) {
		// TODO: more detailed diagnostic
		panic("out of bounds")
	}

	len := j - i

	data := buf.data
	// Make sure we don't create a past-the-end pointer
	if len > 0 {
		data = gpu.UnsafePointerAdd(data, i*buf.type_.Size())
	}

	return IndexBuffer{
		type_: buf.type_,
		data:  data,
		len:   len,
	}
}

// TODO: replace with AsSlice() that returns appropriately typed slice? We might
// still want a (private) function to pull the unsafe pointer to data out of
// IndexBuffer for (*Geometry).AccelConfig
func (buf IndexBuffer) AsByteSlice() gpu.Slice[byte] {
	return gpu.SliceAt(gpu.Pointer[byte](buf.data), buf.len*buf.type_.Size())
}
