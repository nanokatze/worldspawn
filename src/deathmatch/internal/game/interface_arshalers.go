package game

import (
	"fmt"
	"maps"
	"reflect"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"

	"worldspawn/internal/nice"
)

// TODO: move interface arshalers to a different package entirely

/*
func InterfaceJSONMarshaler[T any](types ...reflect.Type) func(*jsontext.Encoder, *T) error {
	return func(enc *jsontext.Encoder, x *T) error {
		v := reflect.ValueOf(*x)
		t := v.Type()

		name, ok := m[t]
		if !ok {
			panic(fmt.Sprintf("trying to marshal an entity of type %v which was not previously registered", t))
		}

		tmp := reflect.New(t)
		tmp.Elem().Set(v)

		if err := enc.WriteToken(jsontext.BeginObject); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String(name)); err != nil {
			return err
		}
		if err := json.MarshalEncode(enc, tmp.Interface()); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.EndObject); err != nil {
			return err
		}
		return nil
	}
}
*/

func InterfaceJSONUnmarshaler[T any](types ...reflect.Type) func(*jsontext.Decoder, *T) error {
	m := maps.Collect(func(yield func(string, reflect.Type) bool) {
		for _, typ := range types {
			yield(typ.Name(), typ)
		}
	})

	return func(dec *jsontext.Decoder, x *T) error {
		if _, err := dec.ReadToken(); err != nil {
			return err
		}

		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		name := tok.String()

		t, ok := m[name]
		if !ok {
			return fmt.Errorf("unknown entity type %s", name)
		}

		data := reflect.New(t)
		if err := json.UnmarshalDecode(dec, data.Interface()); err != nil {
			return err
		}

		if _, err := dec.ReadToken(); err != nil {
			return err
		}

		*x, _ = reflect.TypeAssert[T](data.Elem())
		return nil
	}
}

func InterfaceNiceMarshaler[T any](types ...reflect.Type) func(*nice.Encoder, *T) error {
	m := maps.Collect(func(yield func(reflect.Type, uint64) bool) {
		for idx, typ := range types {
			yield(typ, uint64(idx))
		}
	})

	return func(enc *nice.Encoder, x *T) error {
		data := reflect.ValueOf(*x)
		typ := data.Type()

		id, ok := m[typ]
		if !ok {
			panic(fmt.Sprintf("bad %#v", *x))
		}
		if err := nice.MarshalEncode(enc, &id); err != nil {
			return err
		}

		// TODO: any way we could avoid an alloc?
		tmp := reflect.New(typ)
		tmp.Elem().Set(data)

		return nice.MarshalEncode(enc, tmp.Interface())
	}
}

func InterfaceNiceUnmarshaler[T any](types ...reflect.Type) func(*nice.Decoder, *T) error {
	m := maps.Collect(func(yield func(uint64, reflect.Type) bool) {
		for idx, typ := range types {
			yield(uint64(idx), typ)
		}
	})

	return func(dec *nice.Decoder, x *T) error {
		var id uint64
		if err := nice.UnmarshalDecode(dec, &id); err != nil {
			return err
		}

		typ, ok := m[id]
		if !ok {
			return fmt.Errorf("unknown input command")
		}

		// TODO: any way we could avoid an alloc?
		data := reflect.New(typ)
		if err := nice.UnmarshalDecode(dec, data.Interface()); err != nil {
			return err
		}
		*x, _ = reflect.TypeAssert[T](data.Elem())
		return nil
	}
}
