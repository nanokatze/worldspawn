package gpu

import (
	"bytes"
	"fmt"
	"io"
	"iter"
	"math/bits"
	"reflect"
	"runtime"
	"unsafe"
	"worldspawn/gpu/vk"
)

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

func minify(extent Int3, mipLevel int) Int3 {
	return int3Max(extent.Rsh(int3Splat(mipLevel)), int3Splat(1))
}

func int3Splat(x int) Int3 {
	return Int3{x, x, x}
}

func int3FromVkOffset3D(offset vk.Offset3D) Int3 {
	return Int3{int(offset.X), int(offset.Y), int(offset.Z)}
}

func int3FromVkExtent3D(extent vk.Extent3D) Int3 {
	return Int3{int(extent.Width), int(extent.Height), int(extent.Depth)}
}

func int3ToVkOffset3D(a Int3) vk.Offset3D {
	return vk.Offset3D{
		X: int32(a.X),
		Y: int32(a.Y),
		Z: int32(a.Z),
	}
}

func int3ToVkExtent3D(a Int3) vk.Extent3D {
	return vk.Extent3D{
		Width:  uint32(a.X),
		Height: uint32(a.Y),
		Depth:  uint32(a.Z),
	}
}

func byteSliceToString(s []byte) string {
	return string(s[:bytes.IndexByte(s, 0)])
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
