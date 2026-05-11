package game

import (
	"fmt"
	"maps"
	"reflect"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// TODO: move these into json.go

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

// TODO: make it be a type of parameters?
func InterfaceJSONUnmarshaler[T any](m map[reflect.Type]string) func(*jsontext.Decoder, *T) error {
	m2 := maps.Collect(func(yield func(string, reflect.Type) bool) {
		for k, v := range m {
			yield(v, k)
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

		t, ok := m2[name]
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
