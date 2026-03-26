package sfx

import (
	"io"

	"worldspawn/internal/gmath"
)

// TODO: fold this with the graphics renderer

// Make this opaque?
// TODO: delete this
type Source struct {
	// TODO: do we need multiple format (float32, int16, ...) support? It
	// doesn't seem like we gain much by using narrower formats.
	Samples []float32
}

// TODO: should use something else in place of time.Duration probably

type Instance struct {
	Transform gmath.Affine3f64

	// TODO: stop using this in favor of Source (ReaderAt)
	Samples []float32

	Attenuation float32

	Source io.ReaderAt

	PlayTime int64
}

// TODO: just use graphics renderer's definitions?
type Scene struct {
	// Parent      []int
	// TransformT0 []geometry.TRS3
	// TransformT1 []geometry.TRS3
	// TODO: also velocity, same as in the graphics renderer

	Instance []Instance
}

func sliceLoadOrZero[T any](s []T, i int) T {
	if 0 <= i && i < len(s) {
		return s[i]
	}
	return *new(T)
}

// TODO: separate rendering the kernels from applying them.
// TODO: channels should be IR spheres for HRTF.
// TODO: make sfx.Render output fixed points, and have the caller be responsible
// for interleaving. We should still convert to float32 or whatevs.
func Render(scene *Scene, camera gmath.Vec3f32, now int64, dst []float32, channels, sampleRate int) {
	if channels != 2 {
		panic("not implemented")
	}
	if sampleRate != 48000 {
		panic("not implemented")
	}

	L := len(dst) / channels

	accumulators := make([][]int32, channels)
	for i := 0; i < channels; i++ {
		accumulators[i] = make([]int32, L)
	}

	for _, instance := range scene.Instance {
		src := instance.Samples
		if len(src) == 0 {
			continue
		}

		t0 := now - instance.PlayTime

		dist := gmath.Vec3Convert[float32](instance.Transform.T).Sub(camera).Length()
		contribution := min(10.0/(dist*dist), 1) * instance.Attenuation

		for i := 0; i < L; i++ {
			sample := sliceLoadOrZero(src, int(t0)+i) * contribution

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
