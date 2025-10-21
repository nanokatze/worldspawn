package compiler

import "fmt"

// TODO: rename?
type RewriteContext struct {
	b      *Builder
	values map[*Value]struct{}
	stack  []*Value
}

func (r *RewriteContext) B() *Builder { return r.b }

// TODO: rename, e.g. Value
func (r *RewriteContext) Add2(op Op, typ Type, imm any, args ...*Class) {
	r.Add(r.b.Sea.Value(op, typ, imm, args...))
}

// TODO: rename, e.g. Class
func (r *RewriteContext) Add3(c *Class) {
	for v := range c.Values {
		r.Add(v)
	}
}

func (r *RewriteContext) Add(v *Value) {
	if _, ok := r.values[v]; ok {
		return // we saw this value already
	}
	r.values[v] = struct{}{}
	r.stack = append(r.stack, v)
}

type RewriteRule struct {
	Name    string
	Pattern *Pattern
	Rewrite func(*RewriteContext, *Value) // TODO: I guess we could also eliminate *Value and pass that through *RewriteContext
}

// This handles only 2-ary ops, TODO: teach it to handle n-ary by only swapping
// the first two args?
// TODO: rename to something like CommutativityRule or whatever.
func Commutativity(op Op) RewriteRule {
	return RewriteRule{
		Name: fmt.Sprintf("%v commutativity", op),
		Pattern: &Pattern{
			Op: op,
			Args: []*Pattern{
				{},
				{},
			},
		},
		Rewrite: func(rc *RewriteContext, v *Value) {
			rc.Add2(v.Op(), v.Type(), v.Imm(), v.Args[1], v.Args[0])
		},
	}
}

/*
func Associativity(op Op) RewriteRule {
	return RewriteRule{
		Name: fmt.Sprintf("%v associativity", op),
		Pattern: &Pattern{
			Op: op,
			Args: []*Pattern{
				{},
				{},
			},
		},
		Replace: func(b *Builder, rc *RewriteContext, v *Value) {
		},
	}
}
*/
