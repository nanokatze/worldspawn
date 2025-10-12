package compiler

import "slices"

// TODO: instead of referring to Values by pointers, we should use indices and
// IDs that are contextualized by a particular Sea object. When we add EqValues,
// we'll also need to do the same for EqValues.

type seaKey struct {
	op   Op
	typ  Type
	args [10]*Value // aaaaaaaaa
	imm  any
}

type Sea struct {
	id map[int32]*Value
	m  map[seaKey]*Value // should be seaKey -> id
	// TODO: eidalloc
	vidalloc int32
}

func NewSea() *Sea {
	return &Sea{
		id: make(map[int32]*Value),
		m:  make(map[seaKey]*Value),
	}
}

func (sea *Sea) At(id int32) *Value {
	return sea.id[id]
}

func (sea *Sea) New(op Op, typ Type, args []*Value, imm any) *Value {
	var args_ [10]*Value
	if copy(args_[:], args) != len(args) {
		panic("aaaaaaaa")
	}

	k := seaKey{op, typ, args_, imm}

	v, ok := sea.m[k]
	if !ok {
		id := sea.vidalloc + 1
		sea.vidalloc++
		v = &Value{id, op, typ, slices.Clone(args), imm}
		sea.m[k] = v
		sea.id[id] = v
	}
	return v
}
