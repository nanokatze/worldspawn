package nice

import (
	"fmt"
	"maps"
	"reflect"
)

func InterfaceArshaler[T any](types ...reflect.Type) Arshalers {
	m := maps.Collect(func(yield func(reflect.Type, uint64) bool) {
		for idx, typ := range types {
			yield(typ, uint64(idx))
		}
	})
	m2 := maps.Collect(func(yield func(uint64, reflect.Type) bool) {
		for idx, typ := range types {
			yield(uint64(idx), typ)
		}
	})

	return MakeArshaler(
		func(enc *Encoder, x *T) error {
			data := reflect.ValueOf(*x)
			typ := data.Type()

			id, ok := m[typ]
			if !ok {
				panic(fmt.Sprintf("bad %#v", *x))
			}
			if err := MarshalEncode(enc, &id); err != nil {
				return err
			}

			// TODO: any way we could avoid an alloc?
			tmp := reflect.New(typ)
			tmp.Elem().Set(data)

			return MarshalEncode(enc, tmp.Interface())
		},
		func(dec *Decoder, x *T) error {
			var id uint64
			if err := UnmarshalDecode(dec, &id); err != nil {
				return err
			}

			typ, ok := m2[id]
			if !ok {
				return fmt.Errorf("unknown input command")
			}

			// TODO: any way we could avoid an alloc?
			data := reflect.New(typ)
			if err := UnmarshalDecode(dec, data.Interface()); err != nil {
				return err
			}
			*x, _ = reflect.TypeAssert[T](data.Elem())
			return nil
		})
}
