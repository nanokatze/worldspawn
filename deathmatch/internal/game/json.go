package game

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"unique"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

func float32JSONMarshaler(enc *jsontext.Encoder, x *float32) error {
	tmp := float64(*x)
	return float64JSONMarshaler(enc, &tmp)
}

func float32JSONUnmarshaler(dec *jsontext.Decoder, x *float32) error {
	var tmp float64
	if err := float64JSONUnmarshaler(dec, &tmp); err != nil {
		return err
	}
	*x = float32(tmp)
	return nil
}

func float64JSONMarshaler(enc *jsontext.Encoder, x *float64) error {
	return enc.WriteToken(jsontext.String(jsontext.Float(*x).String()))
}

func float64JSONUnmarshaler(dec *jsontext.Decoder, x *float64) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	if tok.Kind() != '"' {
		// TODO: more informative message
		return fmt.Errorf("expecting string")
	}
	s := tok.String()

	tmp, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	switch {
	case math.IsNaN(tmp) && s != "NaN":
		return strconv.ErrSyntax
	case math.IsInf(tmp, 1) && s != "+Infinity":
		return strconv.ErrSyntax
	case math.IsInf(tmp, -1) && s != "-Infinity":
		return strconv.ErrSyntax
	}
	*x = tmp
	return nil
}

// TODO: if/when https://github.com/golang/go/issues/71664 gets accepted we
// should replace float un/marshalers with WithFormat[floatN]("nonfinite"). We
// should also specify WithFormat[time.Duration]("iso8601") and remove all
// format:iso8601 struct tags.

// TODO: make private and provide methods for de/serializing the scene
var JSONOptions = json.JoinOptions(
	json.StringifyNumbers(true),
	json.WithMarshalers(json.JoinMarshalers(
		json.MarshalToFunc(float32JSONMarshaler),
		json.MarshalToFunc(float64JSONMarshaler),
	)),
	json.WithUnmarshalers(json.JoinUnmarshalers(
		json.UnmarshalFromFunc(func(dec *jsontext.Decoder, v *unique.Handle[string]) error {
			var tmp string
			if err := json.UnmarshalDecode(dec, &tmp); err != nil {
				return nil
			}
			*v = unique.Make(tmp)
			return nil
		}),
		json.UnmarshalFromFunc(float32JSONUnmarshaler),
		json.UnmarshalFromFunc(float64JSONUnmarshaler),

		json.UnmarshalFromFunc(func(dec *jsontext.Decoder, x *ScriptState) error {
			if _, err := dec.ReadToken(); err != nil {
				return err
			}

			tok, err := dec.ReadToken()
			if err != nil {
				return err
			}
			name := tok.String()

			// This sucks, we should maintain a shadow map and only hit the slow
			// path on miss
			var t reflect.Type
			for stateType := range Scripts {
				if stateType.Name() == name {
					t = stateType
					break
				}
			}

			if t == nil {
				return fmt.Errorf("unknown entity type %s", name)
			}

			data := reflect.New(t)
			if err := json.UnmarshalDecode(dec, data.Interface()); err != nil {
				return err
			}

			if _, err := dec.ReadToken(); err != nil {
				return err
			}

			*x, _ = reflect.TypeAssert[ScriptState](data.Elem())
			return nil
		}),
	)))
