package game

import (
	"fmt"
	"maps"
	"reflect"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
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
