package worldspawn

import (
	"fmt"
	"io"
	"reflect"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"

	"worldspawn/experiments/encoding/nice"
)

type Entity any

// TODO: some part of this should be public so that it's up to the callers to
// construct un/marshalers
var entityarshaltab = mkarshaltab()

func registerEntity[T any]() {
	t := reflect.TypeFor[T]()
	entityarshaltab.Register(t.Name(), t)
}

// TODO: it is up to the server and client to implement un/marshalers, we should
// only expose the info necessary for it

func entityJSONUnmarshaler(dec *jsontext.Decoder, entity *Entity) error {
	if _, err := dec.ReadToken(); err != nil { // {
		return err
	}

	tok, err := dec.ReadToken() // string
	if err != nil {
		return err
	}
	name := tok.String()

	t, ok := entityarshaltab.name[name]
	if !ok {
		return fmt.Errorf("unknown entity type %s", name)
	}

	data := reflect.New(t)
	if err := json.UnmarshalDecode(dec, data.Interface()); err != nil {
		return err
	}

	if _, err := dec.ReadToken(); err != nil { // }
		return err
	}

	*entity = data.Elem().Interface()
	return nil
}

// TODO: better panic message

func entityJSONMarshaler(enc *jsontext.Encoder, entity *Entity) error {
	v := reflect.ValueOf(*entity)
	t := v.Type()

	info, ok := entityarshaltab.typ[t]
	if !ok {
		panic(fmt.Sprintf("trying to marshal an entity of type %v which was not previously registered", t))
	}

	tmp := reflect.New(t)
	tmp.Elem().Set(v)

	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String(info.name)); err != nil {
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

func EntityNiceMarshaler(enc *nice.Encoder, entity *Entity) error {
	v := reflect.ValueOf(*entity)
	t := v.Type()

	// TODO: any way we could avoid an alloc?
	tmp := reflect.New(t)
	tmp.Elem().Set(v)

	info, ok := entityarshaltab.typ[t]
	if !ok {
		panic(fmt.Sprintf("trying to marshal an entity of type %v which was not previously registered", t))
	}
	if _, err := enc.Writer().Write(info.hash[:]); err != nil {
		return err
	}

	return nice.MarshalEncode(enc, tmp.Interface())
}

func EntityNiceUnmarshaler(dec *nice.Decoder, entity *Entity) error {
	buf := dec.Scratch(4)
	if _, err := io.ReadFull(dec.Reader(), buf); err != nil {
		return err
	}
	hash := [4]byte(buf)

	t, ok := entityarshaltab.hash[hash]
	if !ok {
		return fmt.Errorf("unknown entity net hash %x", hash)
	}

	// TODO: any way we could avoid an alloc?
	tmp := reflect.New(t)
	if err := nice.UnmarshalDecode(dec, tmp.Interface()); err != nil {
		return err
	}

	*entity = tmp.Elem().Interface()
	return nil
}
