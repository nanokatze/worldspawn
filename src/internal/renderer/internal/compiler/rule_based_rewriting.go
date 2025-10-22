package compiler

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
)

// TODO: rename?
type RewriteContext struct {
	b     *Rewriter
	seen  map[*Value]bool
	stack []*Value
}

func (r *RewriteContext) B() *Rewriter { return r.b }

// TODO: methods for killing the matched value? I guess actually we could make
// seen a map[*Value]bool and set it to false for killed values.

// TODO: rename, e.g. Value
func (r *RewriteContext) Add2(op Op, typ Type, imm any, args ...*Class) {
	r.add(r.b.Sea.value(op, typ, imm, args...))
}

// TODO: rename?
func (r *RewriteContext) Class(c *Class) {
	for _, v := range c.values {
		r.add(v)
	}
}

func (r *RewriteContext) add(v *Value) {
	if _, ok := r.seen[v]; ok {
		return // we saw this value already
	}
	r.seen[v] = true
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
			rc.Add2(v.Op(), v.Type(), v.Imm(), v.Arg(1), v.Arg(0))
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

type Rewriter struct {
	Sea   *Sea
	Rules []RewriteRule
}

// TODO: rename to Build? just Value?
func (b *Rewriter) Value2(op Op, typ Type, imm any, args ...*Class) *Class {
	// TODO: reuse this with a sync.Pool
	rc := &RewriteContext{
		b:    b,
		seen: make(map[*Value]bool),
	}

	rc.Add2(op, typ, imm, args...)

	for len(rc.stack) > 0 {
		v := rc.stack[len(rc.stack)-1]
		rc.stack = rc.stack[:len(rc.stack)-1]

		for _, rule := range b.Rules {
			if rule.Pattern.Match(v) {
				rule.Rewrite(rc, v)
			}
		}
	}

	// TODO: move all of the following code into Sea.class method
	// TODO: if it turns out having the rules kill the current rewritten value
	// doesn't work that well, we can go back to using the seen set and just
	// sort the values.

	for v, keep := range rc.seen {
		if !keep {
			delete(rc.seen, v)
		}
	}

	if c := commonClass(maps.Keys(rc.seen)); c != nil {
		return c
	}

	// kinda gross?
	var values2 []*Value
	var classes2 []*Class
	seenc := make(map[*Class]struct{})
	for v := range rc.seen {
		if c := v.class; c != nil {
			if _, ok := seenc[c]; !ok {
				seenc[c] = struct{}{}
				classes2 = append(classes2, c)
			}
		} else {
			values2 = append(values2, v)
		}
	}

	slices.SortFunc(values2, func(a, b *Value) int { return cmp.Compare(a.ID(), b.ID()) })
	slices.SortFunc(classes2, func(a, b *Class) int { return cmp.Compare(a.ID(), b.ID()) })

	// TODO: move this assertion inside newClass?
	if len(values2) == 0 && len(classes2) < 2 {
		panic("useless")
	}

	return rc.b.Sea.newClass(typ, classes2, values2)
}
