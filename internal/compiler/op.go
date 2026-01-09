package compiler

type Op struct{ id int32 }

// TODO: introduce OpMap[T] for efficiently mapping Op->T
// TODO: we might want an inverse as well, map[string]Op
var opNames = make(map[Op]string)
var opNamesInv = make(map[string]Op)

type Validator func(typ Type, imm any, args ...*Class)

var validators = make(map[Op]Validator)

var nextOpID int32 = 1

// TODO: also provide immediate type? This would let us feed into the parser
// immediately and nicely.
func DefOp(name string, validator Validator) Op {
	id := nextOpID
	nextOpID++
	if id == 0 {
		panic("overflow")
	}

	op := Op{id}
	opNames[op] = name
	opNamesInv[name] = op
	validators[op] = validator
	return op
}

func OpByName(name string) Op { return opNamesInv[name] }

func (op Op) String() string { return opNames[op] }
