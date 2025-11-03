package mc

import "worldspawn/internal/renderer/internal/compiler"

// TODO: ideally we should should follow how MaterialX arranges material
// definition, and MDL to some degree. MDL has a pretty clean way to defining
// materials so we should get inspired by that thing. Actually, MtlX can be
// compiled down to MDL, so maybe we can just model things after MDL?

// TODO: make a kind that we'll instantinate into BSDF, EDF, etc?

type BSDFType struct{}

var BSDF = BSDFType{}

func (BSDFType) String() string { return "BSDF" }

type EDFType struct{}

var EDF = EDFType{}

func (EDFType) String() string { return "EDF" }

/*
type SurfaceType struct{}

func (SurfaceType) String() string { return "Surface" }
*/

var (
	// OpLoadParam = compiler.DefOp("LoadParam", nil)

	// OpLoadAttr = compiler.DefOp("LoadAttr", nil)

	// OpGGX = compiler.DefOp("GGX", nil)

	// OpGeneralizedSchlick = compiler.DefOp("GeneralizedSchlick", nil)

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
