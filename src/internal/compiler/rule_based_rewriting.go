package compiler

import (
	"fmt"
)

type RewriteRule struct {
	Name    string
	Pattern *Pattern
	// TODO: instead of a single *Value, it should be the "binds" that we got
	// from pattern matching
	// TODO: preemptively change match to be a slice?
	Rewrite func(rr *RewriteResult, b *Builder, match *Value)
}

// This handles only 2-ary ops, TODO: teach it to handle n-ary by only swapping
// the first two args? Would be useful for FMA. Alternatively I guess we could
// hand roll commutativity for FMA.
// TODO: rename to something like CommutativityRule or whatever.
func Commutativity(op Op) RewriteRule {
	return RewriteRule{
		Name:    fmt.Sprintf("%v commutativity", op),
		Pattern: &Pattern{Op: op, Args: []*Pattern{{}, {}}},
		Rewrite: func(rr *RewriteResult, b *Builder, v *Value) {
			rr.Add2(v.Op(), v.Type(), v.Imm(), v.Arg(1), v.Arg(0))
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
		Replace: func(rr *RewriteResult, b *Builder, v *Value) {
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
func (rr *RewriteResult) Add2(op Op, typ Type, imm any, args ...*Class) {
	rr.rewriter.Add(rr.sea.value(op, typ, imm, args...))
}

// TODO: rename
func (rr *RewriteResult) Class(c *Class) {
	for v := range c.Values() {
		rr.rewriter.Add(v)
	}
}

// TODO: fold into Add2 and Class functions
func (rr *RewriteResult) Kill(_ *Value) {
	// TODO: just set rr.kill so that we don't hammer the map as much
	rr.rewriter.Kill(rr.rewriting)
}

type ruleBasedRewriter struct {
	values map[*Value]bool
	stack  []*Value
}

func (r *ruleBasedRewriter) Add(v *Value) {
	if _, ok := r.values[v]; ok {
		// We saw this value already
		return
	}
	r.values[v] = true
	r.stack = append(r.stack, v)
}

func (r *ruleBasedRewriter) Kill(v *Value) {
	if _, ok := r.values[v]; !ok {
		panic("haven't seen this value")
	}
	if v.Class() != nil {
		// We don't gain anything by killing this value
		return
	}
	r.values[v] = false
}

func (r *ruleBasedRewriter) Class(b *Builder) *Class {
	r.applyRules(b)

	var typ Type
	for v := range r.values {
		typ = v.Type()
		break
	}
	return b.Sea.class(typ, r.values)
}

func (r *ruleBasedRewriter) applyRules(b *Builder) {
	var rr RewriteResult
	rr.sea = b.Sea
	rr.rewriter = r

	for len(r.stack) > 0 {
		v := r.stack[len(r.stack)-1]
		r.stack = r.stack[:len(r.stack)-1]

		rr.rewriting = v

		for _, rule := range b.Rules {
			if rule.Pattern.Match(v) {
				rule.Rewrite(&rr, b, v)
			}
		}
	}
}
