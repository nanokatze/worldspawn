package cparser

import (
	"unicode"
	"unicode/utf8"
)

type cscanner struct {
	src []byte

	ch    rune
	off   int
	rdOff int
}

func (s *cscanner) Scan() (ctoken, string) {
	s.skipWhitespace()

	ch := s.ch
	switch {
	case ch == '_' || unicode.IsLetter(ch):
		tok := _IDENT
		lit := s.scanIdentifier()
		// TODO: use a map for this
		switch lit {
		case "const":
			tok = _CONST
		case "struct":
			tok = _STRUCT
		}
		return tok, lit

	case unicode.IsDigit(ch):
		return s.scanNumber()

	default:
		s.next()
		switch ch {
		case -1:
			return _EOF, ""
		case '[':
			return _LBRACK, ""
		case ']':
			return _RBRACK, ""
		case '*':
			return _MUL, ""
		case ':':
			return _COLON, ""
		default:
			return _ILLEGAL, string(ch)
		}
	}
}

func (s *cscanner) scanIdentifier() string {
	off := s.off
	for s.ch == '_' || unicode.IsLetter(s.ch) || unicode.IsDigit(s.ch) {
		s.next()
	}
	return string(s.src[off:s.off])
}

func (s *cscanner) scanNumber() (ctoken, string) {
	off := s.off
	for unicode.IsDigit(s.ch) {
		s.next()
	}
	return _INT, string(s.src[off:s.off])
}

func (s *cscanner) skipWhitespace() {
	for s.ch == ' ' || s.ch == '\t' {
		s.next()
	}
}

func (s *cscanner) next() {
	if s.rdOff >= len(s.src) {
		s.off = len(s.src)
		s.ch = -1
		return
	}
	s.off = s.rdOff
	r, w := utf8.DecodeRune(s.src[s.off:])
	s.rdOff += w
	s.ch = r
}
