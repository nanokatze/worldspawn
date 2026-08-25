package protowire

import "testing"

func BenchmarkRecordHeaderAppend(b *testing.B) {
	header := recordHeader{Tag: PackTag(42, TypeBytes), PayloadLen: 69}

	var buf []byte

	for b.Loop() {
		buf = appendRecordHeader(buf[:0], header)
	}
}
