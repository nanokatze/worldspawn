package compiler

// TODO: rename? to Rewriter maybe
type RewriteContext struct {
	sea    *Sea
	values map[*Value]bool
	stack  []*Value
}

// TODO: methods for killing the matched value? I guess actually we could make
// seen a map[*Value]bool and set it to false for killed values.

// TODO: rename, e.g. Value
func (r *RewriteContext) Add2(op Op, typ Type, imm any, args ...*Class) {
	r.add(r.sea.value(op, typ, imm, args...))
}

// TODO: rename?
// TODO: rewrite this to be "safe" (always go through Add2)? Or document the
// requirements that Class must be from the same Sea.
func (r *RewriteContext) Class(c *Class) {
	for _, v := range c.values {
		r.add(v)
	}
}

func (r *RewriteContext) add(v *Value) {
	if _, ok := r.values[v]; ok {
		return // we saw this value already
	}
	r.values[v] = true
	r.stack = append(r.stack, v)
}

// pop a value that has not yet been rewritten
func (r *RewriteContext) pop() *Value {
	if len(r.stack) == 0 {
		return nil
	}
	v := r.stack[len(r.stack)-1]
	r.stack = r.stack[:len(r.stack)-1]
	return v
}
