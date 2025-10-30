// TODO: rename this package to material_compiler or matcomp or matc, and move
// interpreter into matvm?
package mc

import (
	"worldspawn/internal/renderer/internal/compiler"
	"worldspawn/internal/renderer/internal/compiler/core"
)

// TODO: ideally we should should follow how MaterialX arranges material
// definition, and MDL to some degree. MDL has a pretty clean way to defining
// materials so we should get inspired by that thing. Actually, MtlX can be
// compiled down to MDL, so maybe we can just model things after MDL?

type BSDFType struct{}

func (BSDFType) String() string { return "BSDF" }

/*
type EDFType struct{}

func (EDFType) String() string { return "EDF" }

type SurfaceType struct{}

func (SurfaceType) String() string { return "Surface" }
*/

// TODO: actually just kill aliases?

type Builder = compiler.Builder

var (
	OpIConst = core.OpIConst

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
	OpFCeil  = core.OpFCeil

	OpFEqual       = core.OpFEqual
	OpFLess        = core.OpFLess
	OpFLessOrEqual = core.OpFLessOrEqual

	OpDiffuseBSDF = compiler.DefOp("DiffuseBSDF", nil)
	// TODO: if we keep one huge MicrofacetBSDF we'll also want to introduce
	// Fresnel type that we'll pass values of to this op.
	OpMicrofacetBSDF = compiler.DefOp("MicrofacetBSDF", nil)

	// TODO: decide whether to call this uniform (mtlx, osl) or diffuse (mdl)
	OpUniformEDF = compiler.DefOp("UniformEDF", nil)

	OpBSDFReflectance = compiler.DefOp("BSDFReflectance", nil) // BSDF -> color

	OpDFAdd = compiler.DefOp("DFAdd", nil) // T : DF, (T, T) -> DF

	// TODO: rename to e.g. DFTint?
	OpDFScale = compiler.DefOp("DFScale", nil) // T : DF, (T, float32) -> T

	// Even args are weights, odd args are DF values (TODO: swap?)
	// TODO: rename to something else
	OpDFComposition = compiler.DefOp("DFComposition", nil)

	// OpSurface // (BSDF, EDF) -> Surface
)

// Rules for flattening BSDF composition and stuff
// var legalizationRules =
