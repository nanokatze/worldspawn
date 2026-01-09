package core

import "worldspawn/internal/compiler"

type NothingType struct{}

func (NothingType) String() string { return "Nothing" }

var _ compiler.Type = NothingType{}
