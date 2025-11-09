package compiler

// TODO: rename? to Rewriter maybe
type Rewriter struct {
	sea    *Sea
	values map[*Value]bool
	stack  []*Value
}

// TODO: make v implicit?
// TODO: rename
func (rr *Rewriter) Kill(v *Value) {
	if _, ok := rr.values[v]; !ok {
		panic("trying to kill a value we haven't seen")
	}
	rr.values[v] = false
}

// TODO: methods for killing the matched value? I guess actually we could make
// seen a map[*Value]bool and set it to false for killed values.

// TODO: rename, e.g. Value
func (rr *Rewriter) Add2(op Op, typ Type, imm any, args ...*Class) {
	rr.add(rr.sea.value(op, typ, imm, args...))
}

// TODO: rename?
// TODO: rewrite this to be "safe" (always go through Add2)? Or document the
// requirements that Class must be from the same Sea.
func (rr *Rewriter) Class(c *Class) {
	for _, v := range c.values {
		rr.add(v)
	}
}

func (r *Rewriter) add(v *Value) {
	if _, ok := r.values[v]; ok {
		return // we saw this value already
	}
	r.values[v] = true
	r.stack = append(r.stack, v)
}

// pop a value that has not yet been rewritten
func (rr *Rewriter) pop() *Value {
	if len(rr.stack) == 0 {
		return nil
	}
	v := rr.stack[len(rr.stack)-1]
	rr.stack = rr.stack[:len(rr.stack)-1]
	return v
}
