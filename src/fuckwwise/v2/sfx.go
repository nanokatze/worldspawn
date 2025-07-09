package sfxrenderer

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time"
	"unsafe"

	"worldspawn/geometry-go"
	"worldspawn/gpu"

	"gonum.org/v1/gonum/dsp/fourier"
)

type Source struct {
	io.ReaderAt
	Format     int
	Channels   int
	SampleRate int
}

// TODO: add a cached ReaderAt on top
type adapterReaderAt struct {
	mu sync.Mutex
	r  io.ReadSeeker
	// TODO: remember the position we're at so we can avoid a call to Seek
}

var _ io.ReaderAt = (*adapterReaderAt)(nil)

func ReaderAtFromReadSeeker(r io.ReadSeeker) io.ReaderAt {
	if readerAt, ok := r.(io.ReaderAt); ok {
		return readerAt
	}
	return &adapterReaderAt{r: r}
}

func (r *adapterReaderAt) ReadAt(b []byte, off int64) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, err := r.r.Seek(off, io.SeekStart); err != nil {
		return 0, err
	}
	return r.r.Read(b)
}

type Instance struct {
	Transform int

	// TODO: we somehow need things to be persistent, e.g. when the sound
	// stopped, etc. Or maybe that should be up to the user? Perhaps yeah...
	Source Source // should be ReaderAt with other parameters

	// TODO: replace with generic parameters? Not sure if we should use a type
	// other than time.Duration
	PlayTime time.Duration
}

type Scene struct {
	TransformT0 []geometry.TRS3

	Instance []Instance
}

type Microphone struct {
	Pos geometry.Vec3
	Rot geometry.Rot3
	// TODO: must be an IR sphere
	IR []float32
}

// Let's have the renderer just do spatialization and handle waveform generation
// elsewhere, I suppose.

// TODO: we need to specify head information as well, e.g. HRTF/IR.
//
// We'll probably either use SOFA files or synthesize the IR sphere, so let's
// just provide IR sphere one way or another. Each ear will use its own IR
// sphere.

type Renderer struct {
}

func readAtWithZeroBorder(r io.ReaderAt, b []byte, off int64) (int, error) {
	if off < 0 {
		if len(b) < int(-off) {
			clear(b)
			return len(b), nil
		}

		clear(b[:int(-off)])

		n, err := r.ReadAt(b[int(-off):], 0)
		if n >= 0 {
			n += int(-off)
		}
		return n, err
	}

	return r.ReadAt(b, off)
}

func enqueueConvolve(cq *gpu.JobQueue) {
	panic("not implemented")
}

type FFTBuffers struct {
	t    *fourier.FFT
	tmpR []float64
	tmpC []complex128
}

func NewFFTBuffers(n int) *FFTBuffers {
	return &FFTBuffers{
		t:    fourier.NewFFT(n),
		tmpR: make([]float64, n),
		tmpC: make([]complex128, n/2+1),
	}
}

func FFT(x []float32, X []complex64, t *FFTBuffers) {
	if len(t.tmpR) != len(x) {
		panic("bad")
	}
	if len(t.tmpC) != len(X) {
		panic("bad")
	}
	for i := range t.tmpR {
		t.tmpR[i] = float64(x[i])
	}
	t.t.Coefficients(t.tmpC, t.tmpR)
	for i := range t.tmpC {
		X[i] = complex64(t.tmpC[i])
	}
}

func IFFT(X []complex64, x []float32, t *FFTBuffers) {
	if len(t.tmpR) != len(x) {
		panic("bad")
	}
	if len(t.tmpC) != len(X) {
		panic("bad")
	}
	for i := range t.tmpC {
		t.tmpC[i] = complex128(X[i])
	}
	t.t.Sequence(t.tmpR, t.tmpC)
	for i := range t.tmpR {
		x[i] = float32(t.tmpR[i] / float64(t.t.Len()))
	}
}

// TODO: figure out how we should convolve when our audio dst is shorter than
// the kernel length

func Convolve(A, B, AB []float32) {
	if len(A) < len(B) {
		A, B = B, A
	}

	AB = AB[:len(A)+len(B)-1]

	N := len(B)
	N_zp := 2*N - 1

	plan := NewFFTBuffers(N_zp)

	B_zp := make([]float32, N_zp)
	copy(B_zp, B[:N])

	F_B_zp := make([]complex64, N_zp/2+1)
	FFT(B_zp, F_B_zp, plan)

	A_i_zp := make([]float32, N_zp)
	F_A_i_zp := make([]complex64, N_zp/2+1)
	A_i_B_zp := make([]float32, N_zp)
	for i := 0; i < len(A); i += N {
		log.Printf("%.1f%% (%d out of %d)", 100*float32(i)/float32(len(A)), i, len(A))

		A_i := A[i:]
		copy(A_i_zp, A_i[:min(N, len(A_i))])

		FFT(A_i_zp, F_A_i_zp, plan)

		for j := 0; j < N_zp/2+1; j++ {
			F_A_i_zp[j] *= F_B_zp[j]
		}

		IFFT(F_A_i_zp, A_i_B_zp, plan)

		for j := 0; j < len(A_i_B_zp) && i+j < len(AB); j++ {
			AB[i+j] += A_i_B_zp[j]
		}
	}
}

// TODO: provide feedback on errored instances?
func (re *Renderer) Render(
	cq *gpu.JobQueue,
	scene *Scene,
	T0, T1 time.Duration, // TODO: see if this belongs in Scene?
	t float32, // TODO: is this necessary given T0, T1?
	channels []Microphone,
	sampleRate int,
	out []float32) {

	L := len(out) / len(channels)

	acc := make([][]int32, len(channels))
	for i := range acc {
		acc[i] = make([]int32, L)
	}

	// Assumptions: we're playing new samples every Render call, so keeping
	// things in device memory (or alternatively: not involving the host) will
	// only work for certain kinds of effects, which do not change with every
	// call to Render.
	for _, instance := range scene.Instance {
		// xform := scene.TransformT0[instance.Transform]

		src := instance.Source

		t0 := T0 - instance.PlayTime
		off0 := int64(t0) * int64(src.SampleRate) / int64(time.Second) * int64(src.Channels)

		switch src.Format {
		case 1: // SNORM16
		case 2: // FLOAT32
			// TODO: adjust this by pitch etc differences probably. Or maybe
			// delegate that to the effects.
			tmp := make([]float32, L*src.Channels)

			// TODO: apply various delays and stuff here. Tbh we should probably
			// move the switch inside the loop

			// If we get EOF, do nothing. It's up to the user to put this to
			// sleep. At most we can provide feedback on errored instances.
			readAtWithZeroBorder(src, unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(tmp))), len(tmp)*4), off0*4)

			/*
				for j, ch := range channels {
					// dir := ch.Pos.Sub(xform.Translation).NormalizedOr(ch.Dir)

					// TODO: replace attenuation with actually applying the ch.IR

					// attenuation := max(dir.Dot(ch.Dir), 0)

					for i := range L {
						sample := tmp[i]
						acc[j][i] += int32(sample * 32787.0)
					}
				}
			*/
		default:
			panic(fmt.Sprintf("unknown source format %v", src.Format))
		}
	}

	for j := range channels {
		for i := range L {
			out[j*len(channels)+i] = float32(acc[j][i]) / 32767.0
		}
	}
}

// TODO: we should provide a host side resampling helper
