package compiler

// type ValueID int32

// TODO: make the internals private
type Value struct {
	ID   int32 // TODO: type it
	Op   Op
	Type Type
	Args []*Value
	Imm  any
}

/*

type EqValues struct {
	Values map[*Value]struct{}
	// Uses map[*Value]struct{}
}

*/
