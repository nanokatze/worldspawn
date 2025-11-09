package matc

import (
	"worldspawn/internal/renderer/internal/compiler"
	"worldspawn/internal/renderer/internal/compiler/core"
)

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
	// TODO: OpLoadAttr{Scene,Object,Geometry}

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

// TODO: MaterialX geomprop ops can be either integer or float -typed so we'll
// want to be capable of pulling integer attrs probably. But then I guess we
// also need all the integer ops as well...

// TODO: this should return vec4
func LoadAttrObject(b *compiler.Builder, attr string) *compiler.Class {
	return b.Value2(OpInterpreterLoadAttrObject, core.Int32, attr)
}

// TODO: this should return vec4
func LoadAttrGeometry(b *compiler.Builder, attr string) *compiler.Class {
	return b.Value2(OpInterpreterLoadAttrGeometry, core.MakeArrayType(2, core.Int32), attr)
}
