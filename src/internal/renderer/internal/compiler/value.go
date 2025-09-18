package compiler

type Op struct {
	Name string
}

type Value struct {
	Op   *Op
	Type Type
	Args []*Value
	Aux  any
}
