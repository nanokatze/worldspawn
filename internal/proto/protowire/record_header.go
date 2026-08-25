package protowire

const (
	// TODO: explain
	MinRecordHeaderLen = 1

	// TODO: explain
	MaxRecordHeaderLen = 10
)

type recordHeader struct {
	Tag
	PayloadLen int
}

// The minimum number of bytes that this header can be represented with.
func (header recordHeader) MinLen() int {
	n := VarintLen(uint64(header.Tag))
	if header.Type() == TypeBytes {
		n += VarintLen(uint64(header.PayloadLen))
	}
	return n
}

// The maximum number of bytes that this header can be represented with.
func (header recordHeader) MaxLen() int {
	n := MaxVarintLen
	if header.Type() == TypeBytes {
		n += MaxVarintLen
	}
	return n
}

func appendRecordHeader(b []byte, header recordHeader) []byte {
	b = AppendVarint(b, uint64(header.Tag))
	if header.Type() == TypeBytes {
		b = AppendVarint(b, uint64(header.PayloadLen))
	}
	return b
}

func appendPaddedRecordHeader(b []byte, pad int, header recordHeader) []byte {
	if header.Type() == TypeBytes {
		return AppendPaddedVarints(b, pad, uint64(header.Tag), uint64(header.PayloadLen))
	} else {
		return AppendPaddedVarints(b, pad, uint64(header.Tag))
	}
}

func consumeRecordHeader(b []byte) (recordHeader, int) {
	var n int

	tagBits, tagLen := ConsumeVarint(b[n:])
	if tagLen <= 0 {
		return recordHeader{}, -1
	}
	n += tagLen

	tag := Tag(tagBits)

	var payloadLen int
	switch tag.Type() {
	case TypeBytes:
		v, vLen := ConsumeVarint(b[n:])
		if vLen <= 0 {
			return recordHeader{}, -1
		}
		n += vLen

		if v > MaxPayloadLen {
			return recordHeader{}, -1
		}
		payloadLen = int(v)

	case TypeVarint:
		_, payloadLen = ConsumeVarint(b[n:])
		if payloadLen <= 0 {
			return recordHeader{}, -1
		}

	case TypeFixed32:
		payloadLen = 4

	case TypeFixed64:
		payloadLen = 8

	default:
		return recordHeader{}, -1
	}

	return recordHeader{tag, payloadLen}, n
}
