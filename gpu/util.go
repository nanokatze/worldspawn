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

// max(floor(x / 2^p), 1)
func minify(x int, p int) int {
	return max(x>>p, 1)
}

func byteSliceToString(s []byte) string {
	return string(s[:bytes.IndexByte(s, 0)])
}

// TODO: make this generic?
func byteSliceToHostAddressRange(s []byte) vk.HostAddressRangeEXT {
	return vk.HostAddressRangeEXT{
		Address: unsafe.Pointer(unsafe.SliceData(s)),
		Size:    len(s),
	}
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
