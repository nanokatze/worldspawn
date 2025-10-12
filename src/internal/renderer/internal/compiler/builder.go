package compiler

type Builder struct {
	Sea          *Sea
	RewriteRules []RewriteRule
	// Listener     interface{ OnValueCreated(v *Value) }
}

// TODO: rename to something else
func (b *Builder) Value(op Op, typ Type, args []*Value, imm any) *Value {
	v := b.Sea.New(op, typ, args, imm)

	// TODO: apply rewrites

	return v
}

// TODO: make it not take Op and make MakeTuple a standard thing
func BuildTuple(b *Builder, op Op, args ...*Value) *Value {
	elems := make([]Type, len(args))
	for i, a := range args {
		elems[i] = a.Type
	}

	return b.Value(op, MakeTupleType(elems...), args, nil)
}
