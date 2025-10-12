package compiler

type Op struct{ id int32 }

var nextId int32 = 1

// TODO: can we change func(op) to something else so that we can manage the
// storage for the user? And at the same time, allow the user to optimize the
// storage for their own if they need to. I think json/v2 had something like
// that in the works for custom user options.
func DefOp(name string, stuff ...func(op Op)) Op {
	id := nextId
	nextId++

	op := Op{id}

	opName(name)(op)
	for _, f := range stuff {
		f(op)
	}

	return op
}

var opNames = make(map[Op]string)

func opName(name string) func(op Op) { return func(op Op) { opNames[op] = name } }

func (op Op) String() string { return opNames[op] }

var (
	OpMakeTuple    = DefOp("MakeTuple")
	OpTupleExtract = DefOp("TupleExtract")

	OpConst = DefOp("Const")
)
