package nice2

import (
	"encoding/hex"
	"reflect"
	"testing"
)

type myInterface interface{}

func versionedStructArshaler(t reflect.Type, getArshaler Arshalers) Arshaler {
	// version := t.NumField()

	structArshaler := DefaultArshalers(t, getArshaler)

	return Arshaler{
		Size: 8 + structArshaler.Size,
		Marshal: func(heap Heap, p Pointer, v reflect.Value) error {
			panic("no")
		},
		Unmarshal: func(heap Heap, p Pointer, v reflect.Value) error {
			panic("no")
		},
	}
}

func TestXxx(t *testing.T) {
	// var arshalers ArshalerMap

	amap := MakeArshalerMap(
		WithInterfaceArshaler[myInterface]([]reflect.Type{
			reflect.TypeFor[int](),
		}))

	arshalers := amap

	temst := myInterface(42)

	buf, err := Marshal(&temst, arshalers)

	t.Log(err)
	t.Log(hex.Dump(buf))

	if err != nil {
		return
	}

	// buf = buf[:len(buf)-1]

	var boofer myInterface
	t.Log(Unmarshal(buf, &boofer, arshalers))

	t.Log(boofer)
}
