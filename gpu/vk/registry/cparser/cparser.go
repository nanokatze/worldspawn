package cparser

type cparser struct {
	scanner cscanner

	Tok ctoken
	Lit string
}

func newCParser(src []byte) *cparser {
	p := new(cparser)
	p.scanner.src = src
	p.scanner.next()
	p.Next()
	return p
}

func (p *cparser) Next() {
	p.Tok, p.Lit = p.scanner.Scan()
}
