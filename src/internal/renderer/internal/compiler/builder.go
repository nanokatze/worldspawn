package compiler

type Builder struct {
	// TODO: cse, rewrites
}

func (b *Builder) Value(op *Op, typ Type, args []*Value, aux any) *Value {
	return &Value{Op: op, Type: typ, Args: args, Aux: aux}
}
