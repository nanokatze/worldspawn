package compiler

import "unique"

type Op struct{ name unique.Handle[string] }

type Validator func(typ Type, imm any, args ...*Class)

var validators = make(map[Op]Validator)

// TODO: also provide immediate type
// TODO: guard against redefining an op
func DefOp(name string, validator Validator) Op {
	op := Op{unique.Make(name)}
	validators[op] = validator
	return op
}

func (op Op) String() string {
	return op.name.Value()
}
