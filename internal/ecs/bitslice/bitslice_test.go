package bitslice

import (
	"fmt"
	"slices"
	"testing"
)

func TestBoundsChecks(t *testing.T) {
	bs := Make(1)

	for _, test := range []struct {
		name string
		f    func(int)
	}{
		{"Test", func(i int) { bs.Test(i) }},
		{"Set", func(i int) { bs.Set(i) }},
		{"Unset", func(i int) { bs.Unset(i) }},
	} {
		for _, index := range []int{-1, 1} {
			t.Run(fmt.Sprintf("%v(%v)", test.name, index), func(t *testing.T) {
				defer func() {
					if got, want := recover(), (boundsError{x: index, y: 1}); got != want {
						t.Errorf("recover() = %v, want %v", got, want)
					}
				}()

				test.f(index)
			})
		}
	}

	if got, want := bs.words, []word{0}; !slices.Equal(got, want) {
		t.Errorf("bs.bs = %v, want %v", got, want)
	}
	if got, want := bs.ctrs0, []uint32{0}; !slices.Equal(got, want) {
		t.Errorf("bs.ctrs0 = %v, want %v", got, want)
	}
}

// TestSetNextWhileIterating tests that the iterator correctly visits the bits
// that are being set.
func TestSetNextWhileIterating(t *testing.T) {
	N := 10000

	bs := Make(N)
	bs.Set(0)

	var visited []int
	for i := range And(bs) {
		next := i + 1
		if next < N {
			bs.Set(next)
		}

		visited = append(visited, i)
	}

	if got, want := len(visited), N; got != want {
		t.Fatalf("len(visited) = %v, want %v", got, want)
	}

	for i := range visited {
		if got, want := visited[i], i; got != want {
			t.Errorf("visited[%v] = %v, want %v", i, got, want)
		}
	}

	for i, got := range bs.words {
		want := ^word(0)
		if trailing := (i + 1) * wordBits; trailing > N {
			want >>= trailing - N
		}
		if got != want {
			t.Errorf("bs.bs[%v] = 0x%x, want = 0x%x", i, got, want)
		}
	}

	for i, got := range bs.ctrs0 {
		want := uint32(ctr0Bits / wordBits)
		if trailing := (i + 1) * ctr0Bits; trailing > N {
			want -= uint32((trailing - N) / wordBits)
		}
		if got != want {
			t.Errorf("bs.ctrs0[%v] = %v, want = %v", i, got, want)
		}
	}
}

// TestUnsetNextWhileIterating tests that the iterator correctly skips the bits
// that are being unset.
func TestUnsetNextWhileIterating(t *testing.T) {
	N := 10000

	bs := Make(N)
	for i := range N {
		bs.Set(i)
	}

	var visited []int
	for i := range And(bs) {
		next := i + 1
		if next < N {
			if got, want := bs.Unset(next), true; got != want {
				t.Errorf("bs.Unset(%d) = %v, want %v", next, got, want)
			}
		}

		visited = append(visited, i)
	}

	if got, want := len(visited), (N+1)/2; got != want {
		t.Fatalf("len(visited) = %v, want %v", got, want)
	}

	for i := range visited {
		if got, want := visited[i], 2*i; got != want {
			t.Errorf("visited[%v] = %v, want %v", i, got, want)
		}
	}

	for i, got := range bs.words {
		want := word(0)
		for i := 0; i < wordBits; i += 2 {
			want |= 1 << i
		}
		if trailing := (i + 1) * wordBits; trailing > N {
			want >>= trailing - N
		}
		if got != want {
			t.Errorf("bs.bs[%v] = 0x%x, want = 0x%x", i, got, want)
		}
	}

	for i, got := range bs.ctrs0 {
		want := uint32(ctr0Bits / wordBits)
		if trailing := (i + 1) * ctr0Bits; trailing > N {
			want -= uint32((trailing - N) / wordBits)
		}
		if got != want {
			t.Errorf("bs.ctrs0[%v] = %v, want = %v", i, got, want)
		}
	}
}

func BenchmarkUncontendedSetUnset(b *testing.B) {
	N := 10000

	for _, v := range []bool{false, true} {
		b.Run(fmt.Sprintf("Set/%v", v), func(b *testing.B) {
			bs := Make(N)
			for b.Loop() {
				b.StopTimer()
				bitsliceClearDense(bs, v)
				b.StartTimer()

				for i := range N {
					bs.Set(i)
				}
			}

			b.ReportMetric(0, "ns/op")
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*N), "ns/bit")
		})
		b.Run(fmt.Sprintf("Unset/%v", v), func(b *testing.B) {
			bs := Make(N)
			for b.Loop() {
				b.StopTimer()
				bitsliceClearDense(bs, v)
				b.StartTimer()

				for i := range N {
					bs.Unset(i)
				}
			}

			b.ReportMetric(0, "ns/op")
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*N), "ns/bit")
		})
	}
}

func BenchmarkUncontendedIteration(b *testing.B) {
	N := 100000

	for sparsity := 1; sparsity < N; sparsity *= 3 {
		b.Run(fmt.Sprintf("every %v bit set", sparsity), func(b *testing.B) {
			bs := Make(N)
			for i := 0; i < N; i += sparsity {
				bs.Set(i)
			}

			count := 0
			for b.Loop() {
				for range And(bs) {
					count++
				}
			}

			b.ReportMetric(0, "ns/op")
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*N), "ns/bit-of-capacity")
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(count), "ns/bit")
		})
	}
}

func BenchmarkReset(b *testing.B) {
	N := 100000

	for sparsity := 1; sparsity < N; sparsity *= 3 {
		b.Run(fmt.Sprintf("every %d bit set", sparsity), func(b *testing.B) {
			bs := Make(N)
			ones := 0
			for i := 0; i < N; i += sparsity {
				bs.Set(i)
				ones++
			}

			tmp := Make(N)

			for b.Loop() {
				bitsliceCopyDense(tmp, bs)

				tmp.Reset()
			}

			b.ReportMetric(0, "ns/op")
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*ones), "ns/bit")
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*N), "ns/bit-of-capacity")
		})
	}
}

func bitsliceCopyDense(dst, src BitSlice) {
	copy(dst.words, src.words)
	copy(dst.ctrs0, src.ctrs0)
}

func bitsliceClearDense(bs BitSlice, v bool) {
	switch v {
	case false:
		clear(bs.words)
		clear(bs.ctrs0)

	case true:
		// TODO: faster clear
		for i := range bs.len {
			bs.Set(i)
		}
	}
}
