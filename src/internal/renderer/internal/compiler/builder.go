package compiler

type Builder struct {
	Sea          *Sea
	RewriteRules []RewriteRule
}

// TODO: rename to Build? just Value?
func (b *Builder) Value2(op Op, typ Type, imm any, args ...*Class) *Class {
	// TODO: reuse this with a sync.Pool
	rc := &RewriteContext{
		b:      b,
		values: map[*Value]struct{}{},
	}

	rc.Add(b.Sea.Value(op, typ, imm, args...))

	for len(rc.stack) > 0 {
		v := rc.stack[len(rc.stack)-1]
		rc.stack = rc.stack[:len(rc.stack)-1]

		for _, rule := range b.RewriteRules {
			if rule.Pattern.Match(v) {
				rule.Rewrite(rc, v)
			}
		}
	}

	return b.Sea.EquateValues(rc.values)
}
