package protowire

type Type uint8

const (
	TypeBytes   Type = 2
	TypeVarint  Type = 0
	TypeFixed32 Type = 5
	TypeFixed64 Type = 1
)
