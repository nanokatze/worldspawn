package gpu

import (
	"bytes"
	"fmt"
	"io"
	"iter"
	"math/bits"
	"reflect"
	"runtime"
	"sync"
	"unsafe"

	"worldspawn/gpu/vk"
)

func must(err error) {
	// TODO: dump various debug things here if we get to this point?
	if err != nil {
		panic(err)
	}
}

// TODO: rename to something else
func cached[T, U any](f func(T) U) func(T) U {
	var m sync.Map
	return func(k T) U {
		v, ok := m.Load(k)
		if !ok {
			v, _ = m.LoadOrStore(k, f(k))
		}
		return v.(U)
	}
}

func ones32(x uint32) iter.Seq[int] {
	return func(yield func(int) bool) {
		i := 0
		for {
			i += bits.TrailingZeros32(x >> i)
			if i >= 32 {
				break
			}
			if !yield(i) {
				return
			}
			i++
		}
	}
}

// TODO: move elsewhere
func vkBool32(x bool) vk.Bool32 {
	if x {
		return vk.TRUE
	} else {
		return vk.FALSE
	}
}

func int3FromVkOffset3D(from vk.Offset3D) [3]int {
	return [3]int{int(from.X), int(from.Y), int(from.Z)}
}

func int3FromVkExtent3D(from vk.Extent3D) [3]int {
	return [3]int{int(from.Width), int(from.Height), int(from.Depth)}
}

func vkOffset3DFromInt3(from [3]int) vk.Offset3D {
	return vk.Offset3D{X: int32(from[0]), Y: int32(from[1]), Z: int32(from[2])}
}

func vkExtent3DFromInt3(from [3]int) vk.Extent3D {
	return vk.Extent3D{Width: uint32(from[0]), Height: uint32(from[1]), Depth: uint32(from[2])}
}

func minify(x int, level int) int {
	return max(x>>level, 1)
}

func minify3(x [3]int, level int) [3]int {
	return [3]int{
		minify(x[0], level),
		minify(x[1], level),
		minify(x[2], level),
	}
}

func divRoundUp(x, y int) int { return (x + y - 1) / y }

func divRoundUp3(x, y [3]int) [3]int {
	return [3]int{
		divRoundUp(x[0], y[0]),
		divRoundUp(x[1], y[1]),
		divRoundUp(x[2], y[2]),
	}
}

func byteSliceToString(s []byte) string {
	return string(s[:bytes.IndexByte(s, 0)])
}

// TODO: rename
func asbytes(q any) []byte {
	p := reflect.ValueOf(q)
	t := p.Type().Elem()
	return unsafe.Slice((*byte)(p.UnsafePointer()), int(t.Size()))
}

func cstring(s string) *byte {
	a := make([]byte, len(s)+1)
	copy(a, s)
	return unsafe.SliceData(a)
}

func fprintcallers(w io.Writer, callers []uintptr) {
	frames := runtime.CallersFrames(callers)
	for {
		frame, more := frames.Next()
		fmt.Fprintf(w, "%s(...)\n", frame.Function)
		fmt.Fprintf(w, "\t%s:%d\n", frame.File, frame.Line)
		if !more {
			// TODO: print an extra message that there might be more
			// frames
			break
		}
	}
}

// TODO: remove pinned* stuff
func pinned[T any](pinner *runtime.Pinner, p *T) *T {
	pinner.Pin(p)
	return p
}

func pinnedSliceData[T any](pinner *runtime.Pinner, s []T) *T {
	return pinned(pinner, unsafe.SliceData(s))
}

func pinnedCString(pinner *runtime.Pinner, s string) *byte {
	return pinned(pinner, cstring(s))
}

func pinnedCStringSlice(pinner *runtime.Pinner, l []string) **byte {
	ll := make([]*byte, len(l))
	for i, s := range l {
		ll[i] = pinnedCString(pinner, s)
	}
	return pinnedSliceData(pinner, ll)
}
