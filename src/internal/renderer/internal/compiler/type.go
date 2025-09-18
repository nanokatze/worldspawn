package compiler

type Type any

// TODO: move these into different files?
type basicType struct {
	name string
}

func (t *basicType) String() string { return t.name }

var (
	TypeBool   Type = &basicType{name: "Bool"}
	TypeBits8  Type = &basicType{name: "Bits8"}
	TypeBits16 Type = &basicType{name: "Bits16"}
	TypeBits32 Type = &basicType{name: "Bits32"}
)

type TupleType struct {
	// TODO: make it private and provide a Elems() method instead?
	elems []Type
}

// TODO: hash cons tuple types
func MakeTupleType(elems []Type) *TupleType {
	return &TupleType{elems: elems}
}

func (t *TupleType) String() string {
	// TODO: iterate over elems
	return "()"
}

func (t *TupleType) Elems() []Type {
	return t.elems
}
