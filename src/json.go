package worldspawn

import (
	"fmt"
	"strconv"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// TODO: when we move game stuff into game/???/, arshaler code should not be in
// the game's code

// TODO: move this into a subpackage

var JSONOptions = json.JoinOptions(
	json.StringifyNumbers(true),
	json.WithMarshalers(json.JoinMarshalers(
		json.MarshalToFunc(float32JSONMarshaler),
		json.MarshalToFunc(float64JSONMarshaler),
	)),
	json.WithUnmarshalers(json.JoinUnmarshalers(
		json.UnmarshalFromFunc(float32JSONUnmarshaler),
		json.UnmarshalFromFunc(float64JSONUnmarshaler),
	)))

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

// TODO: if this doesn't get inlined into the above, factor the following code
// into yet another function
func float64JSONMarshaler(enc *jsontext.Encoder, x *float64) error {
	// TODO: encode infinities as Infinity rather than Inf
	//
	// TODO: we could use enc.UnusedBuffer() and enc.WriteValue to avoid an
	// allocation, but enc.WriteValue looks to be pretty
	// computationally-intensive. investigate whether we should...
	return enc.WriteToken(jsontext.String(strconv.FormatFloat(*x, 'f', -1, 64)))
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
	*x = tmp
	return nil
}
