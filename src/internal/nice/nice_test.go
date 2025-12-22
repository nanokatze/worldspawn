package nice

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"testing"
)

type codec interface {
	Marshal(buf *bytes.Buffer, in any) error
	Unmarshal(buf *bytes.Reader, out any) error
}

type niceCodec struct {
	enc Encoder
	dec Decoder
}

func (b *niceCodec) Marshal(buf *bytes.Buffer, in any) error {
	b.enc.Reset(buf)
	return MarshalEncode(&b.enc, in)
}

func (b *niceCodec) Unmarshal(buf *bytes.Reader, out any) error {
	b.dec.Reset(buf)
	return UnmarshalDecode(&b.dec, out)
}

func (b *niceCodec) String() string { return "nice" }

type encodingBinaryCodec struct{}

func (*encodingBinaryCodec) Marshal(buf *bytes.Buffer, in any) error {
	return binary.Write(buf, binary.LittleEndian, in)
}

func (*encodingBinaryCodec) Unmarshal(buf *bytes.Reader, out any) error {
	return binary.Read(buf, binary.LittleEndian, out)
}

func (*encodingBinaryCodec) String() string { return "encodingBinary" }

var codecs = []func() codec{
	func() codec { return new(niceCodec) },
	func() codec { return new(encodingBinaryCodec) },
}

var arshalingBenchmarks = []reflect.Type{
	// Contrived cases for measuring overhead
	reflect.TypeFor[struct{}](),
	reflect.TypeFor[[100]struct{}](),

	reflect.TypeFor[int32](),
	reflect.TypeFor[[3]int32](),
	reflect.TypeFor[[100]int32](),
}

func BenchmarkArshaling(b *testing.B) {
	for _, bench := range arshalingBenchmarks {
		for _, newCodec := range codecs {
			codec := newCodec()
			b.Run(fmt.Sprintf("%v/%s", bench, codec), func(b *testing.B) {
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

func TestUnmarshalAllocationAccounting(t *testing.T) {
	budget := 1 << 20

	for _, test := range []struct {
		name string
		make func(n int) any
	}{
		{"map[int]int", func(n int) any {
			m := make(map[int]int)
			for i := range n {
				m[i] = i
			}
			return m
		}},
		{"map[int]struct{}", func(n int) any {
			m := make(map[int]struct{})
			for i := range n {
				m[i] = struct{}{}
			}
			return m
		}},
		{"*[]byte", func(n int) any { tmp := make([]byte, n); return &tmp }},
		{"[]byte", func(n int) any { return make([]byte, n) }},
		{"[][3]byte", func(n int) any { return make([][3]byte, n) }},
		{"[]*byte", func(n int) any {
			tmp := make([]*byte, n)
			for i := range tmp {
				tmp[i] = new(byte)
			}
			return tmp
		}},
		{"string", func(n int) any { return string(make([]byte, n)) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			for n := 1; ; n *= 2 {
				v := test.make(n)

				rv := reflect.ValueOf(v)

				p := reflect.New(rv.Type()).Elem()
				p.Set(rv)

				q := reflect.New(rv.Type()).Elem()

				buf := new(bytes.Buffer)

				if err := MarshalEncode(NewEncoder(buf), p.Addr().Interface()); err != nil {
					t.Fatal(err)
				}

				if err := UnmarshalDecode(NewDecoder(buf, WithBudget(budget)), q.Addr().Interface()); err != nil {
					if oob, ok := err.(*outOfBudgetError); ok {
						t.Logf("%d causes out of budget error (%d needed %d left)", n, oob.n, oob.budget)

						break
					}

					t.Fatal(err)
				}

				if n > budget {
					t.Fatalf("%d is way past the %d budget but no error was produced", n, budget)
				}
			}
		})
	}
}

func TestUnmarshalObjectReplacement(t *testing.T) {
	for _, test := range []struct {
		name string
		new  func() any
		muck func(any)
	}{
		{
			"map[int]int",
			func() any { return map[int]int{0: 1} },
			func(v any) { m := v.(map[int]int); m[1] = 2 },
		},
		{
			"*int",
			func() any { tmp := 1; return &tmp },
			func(v any) { *v.(*int) = 42 },
		},
		{
			"[]int",
			func() any { return []int{1, 0} },
			func(v any) { s := v.([]int); s[1] = 2 },
		},
		// TODO: a more involved test
	} {
		t.Run(test.name, func(t *testing.T) {
			v := test.new()

			rv := reflect.ValueOf(v)

			p := reflect.New(rv.Type()).Elem()
			p.Set(rv)

			q := reflect.New(rv.Type()).Elem()
			q.Set(rv)

			buf := new(bytes.Buffer)

			if err := MarshalEncode(NewEncoder(buf), p.Addr().Interface()); err != nil {
				t.Fatal(err)
			}
			if err := UnmarshalDecode(NewDecoder(buf), q.Addr().Interface()); err != nil {
				t.Fatal(err)
			}

			// Change something about v
			test.muck(v)

			// At this point, p points to v and q should be pointing to a different
			// object, which should've been unaffected by mucking v.
			if reflect.DeepEqual(p.Interface(), q.Interface()) {
				t.Fatalf("object was not replaced")
			}
		})
	}
}
