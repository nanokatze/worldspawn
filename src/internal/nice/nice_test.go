package nice

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

// TODO: lots of tests

// TODO: unify benchmarks under the same top level benchmark

var benchmarks = []struct{}{}

func BenchmarkNiceMarshal(b *testing.B) {
	x := int32(0)
	y := int32(0)

	buf := new(bytes.Buffer)

	enc := NewEncoder(buf)
	dec := NewDecoder(buf)

	// Roundtrip once to ensure we're getting what we expect

	if err := MarshalEncode(enc, &x); err != nil {
		b.Fatal(err)
	}
	if err := UnmarshalDecode(dec, &y); err != nil {
		b.Fatal(err)
	}
	if !reflect.DeepEqual(x, y) {
		// b.Log(x, y)
		b.Fatal("oof")
	}

	b.ReportAllocs()

	for b.Loop() {
		buf.Reset()
		if err := MarshalEncode(enc, &x); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodingBinaryWrite(b *testing.B) {
	x := int32(0)
	y := int32(0)

	buf := new(bytes.Buffer)

	// Roundtrip once to ensure we're getting what we expect

	if err := binary.Write(buf, binary.LittleEndian, &x); err != nil {
		b.Fatal(err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &y); err != nil {
		b.Fatal(err)
	}
	if !reflect.DeepEqual(x, y) {
		// b.Log(x, y)
		b.Fatal("oof")
	}

	b.ReportAllocs()

	for b.Loop() {
		buf.Reset()
		if err := binary.Write(buf, binary.LittleEndian, &x); err != nil {
			b.Fatal(err)
		}
	}
}

func TestDecodeAllocationBudget(t *testing.T) {
	x := new([100000]int32)
	y := new([100000]int32)

	buf := new(bytes.Buffer)

	enc := NewEncoder(buf)
	dec := NewDecoder(buf, WithSizeLimit(1<<20))

	if err := MarshalEncode(enc, &x); err != nil {
		t.Fatal(err)
	}
	if err := UnmarshalDecode(dec, &y); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(x, y) {
		// b.Log(x, y)
		t.Fatal("oof")
	}
}
