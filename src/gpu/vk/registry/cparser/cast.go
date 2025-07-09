package cparser

// TODO: make Node contain a non-public member to make set of Nodes closed?

type Node interface{}

type Name string

type Pointer struct {
	Elem Node
}

type Array struct {
	Elem  Node
	Count Node
}
