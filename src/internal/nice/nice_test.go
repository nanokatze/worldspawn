package nice

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"testing"
)

type codec interface {
	Name() string
	Marshal(buf *bytes.Buffer, in any) error
	Unmarshal(buf *bytes.Reader, out any) error
}

type niceCodec struct {
	enc Encoder
	dec Decoder
}

func (b *niceCodec) Name() string { return "nice" }

func (b *niceCodec) Marshal(buf *bytes.Buffer, in any) error {
	b.enc.Reset(buf)
	return MarshalEncode(&b.enc, in)
}

func (b *niceCodec) Unmarshal(buf *bytes.Reader, out any) error {
	b.dec.Reset(buf)
	return UnmarshalDecode(&b.dec, out)
}

type encodingBinaryCodec struct{}

func (*encodingBinaryCodec) Name() string { return "encodingBinary" }

func (*encodingBinaryCodec) Marshal(buf *bytes.Buffer, in any) error {
	return binary.Write(buf, binary.LittleEndian, in)
}

func (*encodingBinaryCodec) Unmarshal(buf *bytes.Reader, out any) error {
	return binary.Read(buf, binary.LittleEndian, out)
}

var codecs = []func() codec{
	func() codec { return new(niceCodec) },
	func() codec { return new(encodingBinaryCodec) },
}

var codecBenchmarks = []reflect.Type{
	// Contrived cases for measuring overhead
	reflect.TypeFor[struct{}](),
	reflect.TypeFor[[100]struct{}](),

	reflect.TypeFor[int32](),
	reflect.TypeFor[[3]int32](),
	reflect.TypeFor[[100]int32](),
}

func BenchmarkMarshalUnmarshal(b *testing.B) {
	for _, newCodec := range codecs {
		codec := newCodec()
		for _, bench := range codecBenchmarks {
			b.Run(fmt.Sprintf("%s/%v", codec.Name(), bench), func(b *testing.B) {
				want := reflect.New(bench).Interface()
				got := reflect.New(bench).Interface()

				buf := new(bytes.Buffer)

				// Roundtrip once to ensure we're getting what we expect

				if err := codec.Marshal(buf, want); err != nil {
					b.Fatal(err)
				}
				if err := codec.Unmarshal(bytes.NewReader(buf.Bytes()), got); err != nil {
					b.Fatal(err)
				}
				if !reflect.DeepEqual(want, got) {
					b.Fatalf("got %v, want %v", got, want)
				}

				b.Run("Marshal", func(b *testing.B) {
					w := new(bytes.Buffer)
					b.ReportAllocs()
					for b.Loop() {
						w.Reset()
						if err := codec.Marshal(w, want); err != nil {
							b.Fatal(err)
						}
					}
				})

				b.Run("Unmarshal", func(b *testing.B) {
					r := new(bytes.Reader)
					b.ReportAllocs()
					for b.Loop() {
						r.Reset(buf.Bytes())
						if err := codec.Unmarshal(r, got); err != nil {
							b.Fatal(err)
						}
					}
				})
			})
		}
	}
}
