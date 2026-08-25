package protowire

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"testing"
)

func TestVarint(t *testing.T) {
	testData := slices.Collect(func(yield func(uint64) bool) {
		for i := 0; i < 64; i += 7 {
			yield(1<<i - 1)
			yield(1 << i)
		}
		yield(1<<64 - 1)
	})

	for _, v := range testData {
		check := func(t *testing.T, b []byte) {
			want1, want2 := v, len(b)

			got1, got2 := ConsumeVarint(b)
			if got1 != want1 || got2 != want2 {
				t.Errorf("ConsumeVarint(%#v) = (0x%x, %d), want (0x%x, %d)", b, got1, got2, want1, want2)
			}
		}

		t.Run(fmt.Sprintf("Append(...,0x%x)", v), func(t *testing.T) {
			b := AppendVarint(nil, v)
			if got, want := len(b), VarintLen(v); got != want {
				t.Errorf("len(b) = %v, want %v", got, want)
			}
			check(t, b)
		})

		for pad := VarintLen(v); pad <= MaxVarintLen; pad++ {
			t.Run(fmt.Sprintf("AppendPadded(...,%d,0x%x)", pad, v), func(t *testing.T) {
				b := appendPaddedVarint(nil, pad, v)
				if got, want := len(b), pad; got != want {
					t.Errorf("len(b) = %v, want %v", got, want)
				}
				check(t, b)
			})
		}
	}
}

func BenchmarkVarint(b *testing.B) {
	benchmarks := []struct {
		name string
		data func() []uint64
	}{
		{"zeros", func() []uint64 {
			return slices.Collect(func(yield func(uint64) bool) {
				for range 10000 {
					yield(0)
				}
			})
		}},
		{"2^64-1", func() []uint64 {
			return slices.Collect(func(yield func(uint64) bool) {
				for range 10000 {
					yield(math.MaxUint64)
				}
			})
		}},
		{"random", func() []uint64 {
			return slices.Collect(func(yield func(uint64) bool) {
				rnd := rand.NewPCG(1, 2)
				for range 10000 {
					yield(rnd.Uint64())
				}
			})
		}},
	}

	for _, test := range benchmarks {
		b.Run(test.name, func(b *testing.B) {
			testData := test.data()

			b.Run("Append", func(b *testing.B) {
				var buf []byte
				for b.Loop() {
					buf = buf[:0]
					for _, v := range testData {
						buf = AppendVarint(buf, v)
					}
				}
				b.ReportMetric(float64(b.Elapsed())/float64(b.N*len(testData)), "ns/op")
				b.ReportMetric((float64(b.N*len(buf))/1e9)/(float64(b.Elapsed())/1e9), "GB/s")
			})

			b.Run("AppendPadded", func(b *testing.B) {
				var buf []byte
				for b.Loop() {
					buf = buf[:0]
					for _, v := range testData {
						buf = appendPaddedVarint(buf, MaxVarintLen, v)
					}
				}
				b.ReportMetric(float64(b.Elapsed())/float64(b.N*len(testData)), "ns/op")
				b.ReportMetric((float64(b.N*len(buf))/1e9)/(float64(b.Elapsed())/1e9), "GB/s")
			})

			b.Run("Consume", func(b *testing.B) {
				var buf []byte
				for _, v := range testData {
					buf = AppendVarint(buf, v)
				}
				for b.Loop() {
					i := 0
					for i < len(buf) {
						_, n := ConsumeVarint(buf[i:])
						i += n
					}
				}
				b.ReportMetric(float64(b.Elapsed())/float64(b.N*len(testData)), "ns/op")
				b.ReportMetric((float64(b.N*len(buf))/1e9)/(float64(b.Elapsed())/1e9), "GB/s")
			})
		})
	}
}
