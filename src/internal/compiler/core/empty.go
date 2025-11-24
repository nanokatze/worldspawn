package core

import "worldspawn/internal/compiler"

type EmptyType struct{}

func (EmptyType) String() string { return "Empty" }

var _ compiler.Type = EmptyType{}
