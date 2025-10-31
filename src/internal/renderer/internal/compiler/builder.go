package compiler

// TODO: make the internals private and provide a constructor?
type Builder struct {
	Sea   *Sea
	Rules []RewriteRule
}

func (b *Builder) Value2(op Op, typ Type, imm any, args ...*Class) *Class {
	// TODO: reuse this with a sync.Pool
	rc := &RewriteContext{
		sea:    b.Sea,
		values: make(map[*Value]bool),
	}

	rc.Add2(op, typ, imm, args...)

	applyRewriteRules(b, rc)

	// TODO: factor into a method on rc
	return b.Sea.class(typ, rc.values)
}

// TODO: this doesn't really use Builder other than container for sea + rules. I
// guess we could equally make it a method on the Builder. Probably should too
func Rewrite(b *Builder, c *Class) *Class {
	// TODO: see if we can rewrite this non-recursively

	visited := make(map[*Class]*Class)

	var f func(c *Class) *Class
	f = func(c *Class) *Class {
		if x, ok := visited[c]; ok {
			return x
		}

		rc := &RewriteContext{
			sea:    b.Sea,
			values: make(map[*Value]bool),
		}
		for v := range c.Values() {
			// TODO: factor out remapping the args. This also would come in
			// useful in Sea.value.
			args := make([]*Class, len(v.Args()))
			for i := range args {
				args[i] = f(v.Arg(i))
			}
			// Go through value creation path, as we don't know whether it's
			// from the same sea or not.
			rc.Add2(v.Op(), v.Type(), v.Imm(), args...)
		}
		applyRewriteRules(b, rc)

		x := b.Sea.class(c.Type(), rc.values)

		visited[c] = x
		return x
	}
	return f(c)
}
