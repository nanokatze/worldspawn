package draw

import (
	"reflect"
	"runtime"
	"unsafe"
)

func must(err error) {
	if err != nil {
		panic(err)
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
