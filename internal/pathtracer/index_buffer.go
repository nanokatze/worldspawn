package pathtracer

import (
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

type indexType uint8

const (
	indexIdentity indexType = iota
	index8
	index16
	index32
)

func (indexType indexType) Size() int {
	if indexType == indexIdentity {
		return 0
	}
	return 1 << (int(indexType) - 1)
}

// TODO: make public? Rename to ToVkIndexType()?
func (indexType indexType) vkIndexType() vk.IndexType {
	switch indexType {
	case indexIdentity:
		return vk.INDEX_TYPE_NONE_KHR
	case index8:
		return vk.INDEX_TYPE_UINT8
	case index16:
		return vk.INDEX_TYPE_UINT16
	case index32:
		return vk.INDEX_TYPE_UINT32
	default:
		panic("unreachable")
	}
}

type IndexBuffer struct {
	type_ indexType
	data  gpu.UnsafePointer
	len   int
}

func IndexBufferIdentity(len int) IndexBuffer {
	return IndexBuffer{indexIdentity, 0, len}
}

func IndexBufferFromUint8Slice(data gpu.Slice[uint8]) IndexBuffer {
	return IndexBuffer{index8, gpu.UnsafePointer(gpu.SliceData(data)), gpu.SliceLen(data)}
}

func IndexBufferFromUint16Slice(data gpu.Slice[uint16]) IndexBuffer {
	return IndexBuffer{index16, gpu.UnsafePointer(gpu.SliceData(data)), gpu.SliceLen(data)}
}

func IndexBufferFromUint32Slice(data gpu.Slice[uint32]) IndexBuffer {
	return IndexBuffer{index32, gpu.UnsafePointer(gpu.SliceData(data)), gpu.SliceLen(data)}
}

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
