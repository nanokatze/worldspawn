package proto

import (
	"encoding/hex"
	"reflect"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	type S struct {
		A int64
		B string
	}

	b, _ := hex.DecodeString("082a120d68656c6c6f2c20676f72646f6e")

	var v S
	Unmarshal(b, &v)

	t.Log(v)
}

func TestMarshal(t *testing.T) {
	type S struct {
		A int64
		Z string
	}

	v := S{A: 1, Z: "hi"}

	b, err := Append(make([]byte, 0, 1000), &v)
	if err != nil {
		t.Error(err)
	}

	t.Log("hex", hex.EncodeToString(b))

	var v2 S
	if err := Unmarshal(b, &v2); err != nil {
		t.Error(err)
	}

	t.Log(v2)

	if !reflect.DeepEqual(v, v2) {
		t.Error("hmm... bad...")
	}
}

func BenchmarkMarshal(b *testing.B) {
	type S struct {
		A int64
		B struct {
			Y string
		}
		Z string
	}

	var v S
	v.A = 42
	v.B.Y = "ab"
	v.Z = "hello, gordon"

	/*
		buf, err := Append(make, &v)
		b.Log(hex.Dump(buf))
		if err != nil {
			b.Fatal(err)
		}
	*/

	buf := make([]byte, 0, 10000)

	b.ReportAllocs()

	for b.Loop() {
		buf, _ = Append(buf[:0], &v)
	}

	_ = buf
}
