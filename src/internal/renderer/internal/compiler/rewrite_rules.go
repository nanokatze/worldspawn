package compiler

import (
	"fmt"
)

type RewriteRule struct {
	Name    string
	Pattern *Pattern
	Rewrite func(*Builder, *RewriteContext, *Value)
}

// This handles only 2-ary ops, TODO: teach it to handle n-ary by only swapping
// the first two args?
// TODO: rename to something like CommutativityRule or whatever.
func Commutativity(op Op) RewriteRule {
	return RewriteRule{
		Name: fmt.Sprintf("%v commutativity", op),
		Pattern: &Pattern{
			Op:   op,
			Args: []*Pattern{{}, {}},
		},
		Rewrite: func(b *Builder, rc *RewriteContext, v *Value) {
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

// TODO: make this public?
func applyRewriteRules(b *Builder, rc *RewriteContext) {
	for {
		v := rc.pop()
		if v == nil {
			break
		}
		for _, rule := range b.Rules {
			if rule.Pattern.Match(v) {
				rule.Rewrite(b, rc, v)
			}
		}
	}
}
