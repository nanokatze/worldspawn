package cparser

// TODO: move this to a subpackage?

// TODO: handle bitfields. As of now, AccelerationStructureInstanceKHR is broken
// because we don't handle bitfields.
func ParseDecl(decl []byte) (string, Node) {
	p := newCParser(decl)

	declSpec := parseDeclSpec(p)

	ty := declSpec

DeclaratorLoop:
	for {
		switch p.Tok {
		case _MUL:
			ty = Pointer{Elem: ty}
		case _CONST:
			// do nothing
		default:
			break DeclaratorLoop
		}
		p.Next()
	}

	name := p.Lit
	p.Next()

	ty = parseArrayStuff(p, ty)

	return name, ty
}

// TODO: rename
func parseArrayStuff(p *cparser, ty Node) Node {
	if p.Tok == _LBRACK {
		p.Next()

		count := Name(p.Lit)
		p.Next()

		if p.Tok == _RBRACK {
			p.Next()
		}

		ty = Array{Elem: parseArrayStuff(p, ty), Count: count}
	}

	return ty
}

func parseDeclSpec(p *cparser) Node {
	for {
		switch p.Tok {
		case _IDENT:
			lit := p.Lit
			p.Next()
			return Name(lit)
		case _CONST, _STRUCT:
			// nothing to do
		default:
			return nil
		}
		p.Next()
	}
}
