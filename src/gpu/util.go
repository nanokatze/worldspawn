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

func splat3[T any](x T) [3]T {
	return [3]T{x, x, x}
}

func minify3(extent [3]int, level int) [3]int {
	return int3(extent).Rsh(splat3(level)).Max(splat3(1))
}

// TODO: change this to be in terms of [3]int?
func int3FromVkOffset3D(offset vk.Offset3D) [3]int {
	return [3]int{int(offset.X), int(offset.Y), int(offset.Z)}
}

func int3FromVkExtent3D(extent vk.Extent3D) [3]int {
	return [3]int{int(extent.Width), int(extent.Height), int(extent.Depth)}
}

func vkOffset3DFromInt3(a [3]int) vk.Offset3D {
	return vk.Offset3D{X: int32(a[0]), Y: int32(a[1]), Z: int32(a[2])}
}

func vkExtent3DFromInt3(a [3]int) vk.Extent3D {
	return vk.Extent3D{Width: uint32(a[0]), Height: uint32(a[1]), Depth: uint32(a[2])}
}

func int3DivRoundUp(a, b [3]int) [3]int {
	return int3(a).Add(b).Sub(splat3(1)).Div(b)
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
