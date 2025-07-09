package cparser

type ctoken int

const (
	_ILLEGAL ctoken = iota
	_EOF

	_IDENT
	_INT
	_FLOAT

	_MUL

	_LBRACK

	_RBRACK
	_COLON

	_CONST

	_ENUM
	_STRUCT
	_UNION

	_VOID
	_CHAR
)
