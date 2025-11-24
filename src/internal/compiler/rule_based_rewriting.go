package compiler

import (
	"fmt"
)

type RewriteRule struct {
	Name    string
	Pattern *Pattern
	// TODO: instead of a single *Value, it should be the "binds" that we got
	// from pattern matching
	// TODO: swap *RewriteResult and *Builder places?
	// TODO: preemptively change match to be a slice?
	Rewrite func(b *Builder, r *RewriteResult, match *Value)
}

// This handles only 2-ary ops, TODO: teach it to handle n-ary by only swapping
// the first two args? Would be useful for FMA. Alternatively I guess we could
// hand roll commutativity for FMA.
// TODO: rename to something like CommutativityRule or whatever.
func Commutativity(op Op) RewriteRule {
	return RewriteRule{
		Name:    fmt.Sprintf("%v commutativity", op),
		Pattern: &Pattern{Op: op, Args: []*Pattern{{}, {}}},
		Rewrite: func(b *Builder, r *RewriteResult, v *Value) {
			r.Add2(v.Op(), v.Type(), v.Imm(), v.Arg(1), v.Arg(0))
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

// TODO: rename? This is more like RewriteResultBuilder
type RewriteResult struct {
	sea       *Sea
	rewriter  *ruleBasedRewriter
	rewriting *Value
}

// TODO: rename
func (r *RewriteResult) Add2(op Op, typ Type, imm any, args ...*Class) {
	r.rewriter.add(r.sea.value(op, typ, imm, args...))
}

func (r *RewriteResult) Class(c *Class) {
	for v := range c.Values() {
		r.rewriter.add(v)
	}
}

// TODO: fold into Add2 and Class functions so as to avoid killing all values
// and ending up with an empty class.
func (r *RewriteResult) Kill(_ *Value) {
	if _, ok := r.rewriter.values[r.rewriting]; !ok {
		panic("we haven't seen this value, hello?")
	}
	r.rewriter.values[r.rewriting] = false
}

type ruleBasedRewriter struct {
	values map[*Value]bool
	stack  []*Value
}

func (r *ruleBasedRewriter) add(v *Value) {
	if _, ok := r.values[v]; ok {
		// We saw this value already
		return
	}
	r.values[v] = true
	r.stack = append(r.stack, v)
}

// pop a value that has not yet been rewritten
// TODO: rename to e.g. popValueToRewrite?
func (rr *ruleBasedRewriter) pop() *Value {
	if len(rr.stack) == 0 {
		return nil
	}
	v := rr.stack[len(rr.stack)-1]
	rr.stack = rr.stack[:len(rr.stack)-1]
	return v
}

// TODO: make this a method and also immediately make this spit out *Class)
func (rc *ruleBasedRewriter) applyRules(b *Builder) {
	var r RewriteResult
	r.sea = b.Sea
	r.rewriter = rc
	for {
		v := rc.pop()
		if v == nil {
			break
		}
		r.rewriting = v
		for _, rule := range b.Rules {
			if rule.Pattern.Match(v) {
				rule.Rewrite(b, &r, v)
			}
		}
	}
}
