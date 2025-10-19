// TODO: rename this package to material_compiler or matcomp or matc, and move
// interpreter into matvm?
package material

import (
	"fmt"

	"worldspawn/internal/renderer/internal/compiler"
)

// type BxDFType struct{}

/*
type FloatType struct{ E, M int8 }

func (t FloatType) String() string { return fmt.Sprintf("Float[%d, %d]", t.E, t.M) }

func (t FloatType) BitsType() BitsType {
	return MakeBitsType(1 + t.E + t.M)
}
*/

// TODO: replace with a specialized material builder
// TODO: new idea: hide compiler.Op, compiler.Value, and material Builder should
// expose a set of blessed ops on its own. We can also move that builder to the
// base package.

type Builder = compiler.Builder

var (
	OpConst = compiler.OpConst

	OpMakeArray    = compiler.OpMakeArray
	OpArrayExtract = compiler.OpArrayExtract

	// TODO: introduce float type which will specify e, m
	OpFAdd = compiler.DefOp("FAdd", nil)
	OpFSub = compiler.DefOp("FSub", nil)
	OpFMul = compiler.DefOp("FMul", nil)
	OpFDiv = compiler.DefOp("FDiv", nil)
	OpFMin = compiler.DefOp("FMin", nil)
	OpFMax = compiler.DefOp("FMax", nil)

	OpFFloor = compiler.DefOp("FFloor", nil)

	OpFEqual       = compiler.DefOp("FEqual", nil)
	OpFLessOrEqual = compiler.DefOp("FLessOrEqual", nil)

	OpCondSelect = compiler.DefOp("CondSelect", nil)
)

/*
var (
	// OpSampleTexture = compiler.DefOp("SampleTexture")

	OpXDFAdd   = compiler.DefOp("XDFAdd")
	OpXDFScale = compiler.DefOp("XDFScale")
)
*/

// TODO: I guess we could also make a universal OpBSDF, but that is kinda sucky
var (
// TODO: make EDF/BSDF the suffix?
// OpEDFUniform  = compiler.DefOp("EDFUniform")
// OpBSDFDiffuse = compiler.DefOp("BSDFDiffuse")
)

type regRange struct{ I, N int }

func (rr regRange) String() string {
	// if rr.n < 1 {
	// 	panic("wat")
	// }
	if rr.N == 1 {
		return fmt.Sprintf("r%d", rr.I)
	}
	return fmt.Sprintf("r[%d:%d]", rr.I, rr.I+rr.N-1)
}
