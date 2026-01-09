package compiler

import "strings"

// TODO: implement binding. That's actually not very trivial with egraphs at
// play

// TODO: make mini-DSL for patterns? We don't need it be in a separate language,
// just CompilePattern(string) *Pattern or something

type Pattern struct {
	Op      Op
	Imm     any        // TODO: implement
	Args    []*Pattern // TODO: make it non-pointer
	ArgsDDD bool
	// Bind    int
}

func (p *Pattern) MatchClass(c *Class) bool {
	for n := range c.Values() {
		if p.Match(n) {
			return true
		}
	}
	return false
}

func (p *Pattern) Match(v *Value) bool {
	if p.Op != (Op{}) {
		if v.Op() != p.Op {
			return false
		}
		if !p.ArgsDDD && len(v.Args()) != len(p.Args) ||
			p.ArgsDDD && len(v.Args()) < len(p.Args) {
			return false
		}
		for i, r := range p.Args {
			a := v.Arg(i)
			if !r.MatchClass(a) {
				return false
			}
		}
	}
	return true
}

func (p *Pattern) String() string {
	pp := patternPrinter{}
	pp.print(p)
	return pp.String()
}

// TODO: deuglify pattern printer code
type patternPrinter struct {
	tokens []string
}

// TODO: rename to addToken? printToken? putToken? idk
func (pp *patternPrinter) writeToken(tok string) {
	if len(pp.tokens) >= 2 && pp.tokens[len(pp.tokens)-2] == "(" && tok == ")" {
		pp.tokens[len(pp.tokens)-2] = pp.tokens[len(pp.tokens)-1]
		pp.tokens = pp.tokens[:len(pp.tokens)-1]
		return
	}
	pp.tokens = append(pp.tokens, tok)
}

func (pp *patternPrinter) print(p *Pattern) {
	pp.writeToken("(")
	if p.Op == (Op{}) {
		pp.writeToken("_")
	} else {
		pp.writeToken(p.Op.String())
	}
	// if p.Imm != nil { }
	for _, a := range p.Args {
		pp.print(a)
	}
	if p.ArgsDDD {
		pp.writeToken("...")
	}
	pp.writeToken(")")
}

// TODO: split this into two tables: needSpaceAfter, needSpaceBefore?
func needBlankBetweenTokens(a, b string) bool {
	if a == "" {
		return false
	}
	if a == "(" {
		return false
	}
	if b == ")" {
		return false
	}
	return true
}

func (pp *patternPrinter) String() string {
	var b strings.Builder
	var lastTok string
	for _, tok := range pp.tokens {
		if needBlankBetweenTokens(lastTok, tok) {
			b.WriteByte(' ')
		}
		b.WriteString(tok)
		lastTok = tok
	}
	return b.String()
}
