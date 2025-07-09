package sfx

import (
	"math"
	"time"

	"worldspawn/geometry-go"
)

const (
	SNORM16 = 1
	FLOAT32 = 2
)

// TODO: fold this with the graphics renderer

// Make this opaque?
type Source struct {
	// TODO: do we need multiple format (float32, int16, ...) support? It
	// doesn't seem like we gain much by using narrower formats.
	Samples    []float32
	SampleRate int // TODO: should be FastDivisor32 as we'll be using fixed point.
	// TODO: do we want a time offset here? Alternatively we could resample on
	// upload if our sample falls in-between samples.
}

// http://yehar.com/blog/wp-content/uploads/2009/08/deip.pdf
func interp(yminus1, y0, y1, y2, t float32) float32 {
	return (yminus1*-((t-3)*t+2)+
		y0*3*((t-2)*t-1)+
		y1*-3*((t-1)*t-2)+
		y2*(t*t-1))*(t/6) + y0
}

func sloadzp(s []float32, i int) float32 {
	if i < 0 || len(s) <= i {
		return 0
	}
	return s[i]
}

// TODO: replace t with fixed point nanoseconds
// TODO: make private?
func (s *Source) Sample(t float64) float32 {
	t *= float64(s.SampleRate)

	iminus1 := int(t) - 1
	i0 := int(t) + 0
	i1 := int(t) + 1
	i2 := int(t) + 2

	return interp(
		sloadzp(s.Samples, iminus1),
		sloadzp(s.Samples, i0),
		sloadzp(s.Samples, i1),
		sloadzp(s.Samples, i2),
		float32(t-math.Floor(t)))
}

// TODO: should use something else in place of time.Duration probably

type Instance struct {
	Transform int

	Source *Source

	PlayTime time.Duration
}

// TODO: just use graphics renderer's definitions?
type Scene struct {
	Parent      []int
	TransformT0 []geometry.TRS3
	TransformT1 []geometry.TRS3
	// TODO: also velocity, same as in the graphics renderer

	Instance []Instance
	// TODO: output sinks
}

// TODO: separate rendering the kernels from applying them.
// TODO: channels should be IR spheres for HRTF.
// TODO: make sfx.Render output fixed points, and have the caller be responsible
// for interleaving. We should still convert to float32 or whatevs.
func Render(scene *Scene, now time.Duration, dst []float32, channels, sampleRate int) {
	if sampleRate != 48000 {
		panic("not implemented")
	}
	if channels != 2 {
		panic("not implemented")
	}

	L := len(dst) / channels

	accumulators := make([][]int32, channels)
	for i := 0; i < channels; i++ {
		accumulators[i] = make([]int32, L)
	}

	for _, instance := range scene.Instance {
		src := instance.Source

		t0 := now - instance.PlayTime

		for i := 0; i < L; i++ {
			sample := src.Sample(float64(t0)/1e9 + (float64(i) / float64(sampleRate)))

			for j := 0; j < channels; j++ {
				accumulators[j][i] += int32(sample * 32787.0)
			}
		}
	}

	// Convert our 16.16 fixed point into float32.

	for channel, channelSamples := range accumulators {
		for i, sample := range channelSamples {
			dst[channel+i*channels] = float32(sample) / 32787.0
		}
	}
}
