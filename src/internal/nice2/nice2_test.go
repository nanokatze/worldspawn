package nice2

import (
	"encoding/hex"
	"reflect"
	"testing"
)

type myInterface interface{}

func TestXxx(t *testing.T) {
	// var arshalers ArshalerMap

	arshalers := MakeArshalerMap(
		WithTypedArshaler(func(getArshaler ArshalerGetter) TypedArshaler[myInterface] {
			return TypedArshaler[myInterface]{
				Size: 8,
				Marshal: func(heap Heap, p Pointer, v *myInterface) error {
					mfw := getArshaler(reflect.TypeFor[int]())
					tmp := (*v).(int)
					return mfw.Marshal(heap, p, reflect.ValueOf(&tmp).Elem())
				},
				Unmarshal: func(heap Heap, p Pointer, v *myInterface) error {
					mfw := getArshaler(reflect.TypeFor[int]())
					var tmp int
					err := mfw.Unmarshal(heap, p, reflect.ValueOf(&tmp).Elem())
					*v = myInterface(tmp)
					return err
				},
			}
		}))

	temst := myInterface(42)

	buf, err := Marshal(&temst, arshalers.Get)

	t.Log(err)
	t.Log(hex.Dump(buf))

	if err != nil {
		return
	}

	// buf = buf[:len(buf)-1]

	var boofer myInterface
	t.Log(Unmarshal(buf, &boofer, arshalers.Get))

	t.Log(boofer)
}
