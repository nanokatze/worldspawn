package core

import (
	"fmt"

	"worldspawn/internal/compiler"
)

// TODO: introduce an op to convert between FloatType and Int
type FloatType struct {
	E int
	M int
}

func (t FloatType) String() string { return fmt.Sprintf("Float[%d,%d]", t.E, t.M) }

var FloatE8M23 compiler.Type = FloatType{8, 23}

var (
	OpFAdd = defOp("FAdd", nil)
	OpFSub = defOp("FSub", nil)
	OpFMul = defOp("FMul", nil)
	OpFDiv = defOp("FDiv", nil)

	OpFMin = defOp("FMin", nil)
	OpFMax = defOp("FMax", nil)

	OpFFloor = defOp("FFloor", nil)
	OpFCeil  = defOp("FCeil", nil)

	OpFEqual       = defOp("FEqual", nil)
	OpFLess        = defOp("FLess", nil)
	OpFLessOrEqual = defOp("FLessOrEqual", nil)
)

func init() {
	Rules = append(Rules,
		compiler.Commutativity(OpFAdd),
		compiler.Commutativity(OpFMul),

		compiler.Commutativity(OpFMin),
		compiler.Commutativity(OpFMax),

		compiler.Commutativity(OpFEqual),
	)
}
