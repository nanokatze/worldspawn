package game

import (
	"fmt"
	"strconv"

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
	// TODO: encode infinities as Infinity rather than Inf
	//
	// TODO: we could use enc.UnusedBuffer() and enc.WriteValue to avoid an
	// allocation
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
