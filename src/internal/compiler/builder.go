package compiler

// TODO: make the internals private and provide a constructor?
type Builder struct {
	Sea   *Sea
	Rules []RewriteRule
}

func (b *Builder) Value2(op Op, typ Type, imm any, args ...*Class) *Class {
	r := &ruleBasedRewriter{
		values: make(map[*Value]bool),
	}
	r.Add(b.Sea.value(op, typ, imm, args...))
	return r.Class(b)
}

func Rewrite(b *Builder, c *Class) *Class {
	// TODO: see if we can rewrite this non-recursively

	visited := make(map[*Class]*Class)

	var f func(c *Class) *Class
	f = func(c *Class) *Class {
		if x, ok := visited[c]; ok {
			return x
		}

		r := &ruleBasedRewriter{
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
			r.Add(b.Sea.value(v.Op(), v.Type(), v.Imm(), args...))
		}
		x := r.Class(b)

		visited[c] = x
		return x
	}
	return f(c)
}
