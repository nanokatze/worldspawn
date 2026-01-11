package core

import "worldspawn/internal/compiler"

// TODO: each float op should describe how the float should be interpreted and
// other things (like the result not ever being nan etc)

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
