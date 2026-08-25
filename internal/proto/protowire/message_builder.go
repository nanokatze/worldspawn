package protowire

import (
	"slices"
	"unsafe"
)

type MessageBuilder []byte

func (b *MessageBuilder) ScratchBuffer(reserve int) []byte {
	reserve = min(max(reserve, 1), MaxRecordHeaderLen)

	scratch := (*b)[len(*b):]
	if reserve <= cap(scratch) {
		return scratch[reserve:reserve]
	}
	return nil
}

func (b *MessageBuilder) AppendRecord(r Record) {
	const bigPayload = 256

	header := r.header()

	scratchOff := len(*b)
	payloadOff := ptrdiff(r.Payload, *b)

	if payloadOff >= scratchOff {
		// The payload is in the scratch buffer.

		hole := payloadOff - scratchOff

		end := payloadOff + len(r.Payload)

		if hole < header.MinLen() {
			// The hole is not large enough to fit the header. Shift the payload
			// and write the header.

			tmp := make([]byte, 0, MaxRecordHeaderLen)
			tmp = appendRecordHeader(tmp, header)
			*b = slices.Replace((*b)[:end], scratchOff, payloadOff, tmp...)
			return
		}

		if len(r.Payload) >= bigPayload && hole <= header.MaxLen() {
			// Write the header with padding so that it fills the entire hole.

			*b = appendPaddedRecordHeader(*b, hole, header)
			*b = (*b)[:end]
			return
		}
	}

	// The general case: write the header, followed by payload. Writing the
	// header won't clobber the payload as this case has been already handled
	// above.

	*b = appendRecordHeader(*b, header)
	*b = append(*b, r.Payload...)
}

func ptrdiff(a, b []byte) int {
	p := uintptr(unsafe.Pointer(unsafe.SliceData(a)))
	q := uintptr(unsafe.Pointer(unsafe.SliceData(b)))
	d := int(p - q)
	if d >= 0 && d <= cap(b) {
		return d
	}
	return -1
}
