// TODO: rename this package to material_compiler or matcomp or matc, and move
// interpreter into matvm?
package mc

import (
	"worldspawn/internal/renderer/internal/compiler"
	"worldspawn/internal/renderer/internal/compiler/core"
)

// type BSDFType struct{}

/*
type FloatType struct{ E, M int8 }

func (t FloatType) String() string { return fmt.Sprintf("Float[%d, %d]", t.E, t.M) }

func (t FloatType) BitsType() BitsType {
	return MakeBitsType(1 + t.E + t.M)
}

type FloatControls struct {
	Finite bool
	// stuff
}
*/

// TODO: replace with a specialized material builder
// TODO: new idea: hide compiler.Op, compiler.Value, and material Builder should
// expose a set of blessed ops on its own. We can also move that builder to the
// base package.

// TODO: actually just kill aliases?

type Builder = compiler.Rewriter

var (
	OpConst = core.OpConst

	OpMakeArray    = core.OpMakeArray
	OpArrayExtract = core.OpArrayExtract

	OpCondSelect = core.OpCondSelect

	// TODO: move these to float subpackage of compiler base package
	OpFAdd = core.OpFAdd
	OpFSub = core.OpFSub
	OpFMul = core.OpFMul
	OpFDiv = core.OpFDiv

	OpFMin = core.OpFMin
	OpFMax = core.OpFMax

	OpFFloor = core.OpFFloor

	OpFEqual       = core.OpFEqual
	OpFLess        = core.OpFLess
	OpFLessOrEqual = core.OpFLessOrEqual

	OpBSDFAdd = compiler.DefOp("BSDFAdd", nil)
	// OpBSDFScale = compiler.DefOp("BSDFScale", nil)

	OpBSDFAlbedo = compiler.DefOp("BSDFAlbedo", nil)
)

// TODO: I guess we could also make a universal OpBSDF, but that is kinda sucky
var (
// TODO: make EDF/BSDF the suffix?
// OpEDFUniform  = compiler.DefOp("EDFUniform")
// OpBSDFDiffuse = compiler.DefOp("BSDFDiffuse")
)
