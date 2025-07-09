package parallel

import (
	"image"
	"image/png"
	"iter"
	"math"
	"os"
	"testing"
)

func slices_All[Slice ~[]E, E any](s Slice) iter.Seq2[int, E] {
	return func(yield func(int, E) bool) {
		for i, v := range s {
			if !yield(i, v) {
				return
			}
		}
	}
}

func BenchmarkFor(b *testing.B) {
	f, err := os.Open("test.png")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()

	imgUnknown, err := png.Decode(f)
	if err != nil {
		b.Fatal(err)
	}
	img := imgUnknown.(*image.Gray)

	img2 := image.NewGray(img.Rect)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		For(slices_All(img.Pix), 1000,
			func(k int, v uint8) {
				x := unorm8decode(v)

				for i := 0; i < 3; i++ {
					x = cos32(x)
				}

				img2.Pix[k] = unorm8encode(x)
			})
	}
}

func unorm8decode(x uint8) float32 {
	return float32(x) / 255
}

func unorm8encode(x float32) uint8 {
	return uint8(min(max(x, 0), 1)*255 + 0.5)
}

func cos32(x float32) float32 {
	return float32(math.Cos(float64(x)))
}
