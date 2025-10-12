package compiler

import (
	"slices"
	"strconv"
	"strings"
)

type Type interface {
	String() string
}

type TupleType struct {
	// TODO: make it private and provide a Elems() method instead?
	elems []Type
}

// TODO: hash cons tuple types
func MakeTupleType(elems ...Type) *TupleType {
	return &TupleType{slices.Clone(elems)}
}

func (t *TupleType) String() string {
	elems := make([]string, len(t.elems))
	for i, e := range t.elems {
		elems[i] = e.String()
	}

	return "(" + strings.Join(elems, ", ") + ")"
}

func (t *TupleType) Elems() []Type {
	return t.elems
}

type BitsType struct{ N int64 }

func (t BitsType) String() string { return "Bits[" + strconv.FormatInt(t.N, 10) + "]" }

var (
	Bits1  Type = BitsType{1}
	Bits8  Type = BitsType{8}
	Bits16 Type = BitsType{16}
	Bits32 Type = BitsType{32}
)
