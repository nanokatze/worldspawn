package matc

import (
	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
)

// TODO: make a kind that we'll instantinate into BSDF, EDF, etc?

type BSDFType struct{}

func (BSDFType) String() string { return "BSDF" }

type EDFType struct{}

func (EDFType) String() string { return "EDF" }

// Surface is a pair of BSDF and EDF.
// TODO: rename? To e.g. BSDFEDFPair.
type SurfaceType struct{}

func (SurfaceType) String() string { return "Surface" }

type MaterialType struct{}

func (MaterialType) String() string { return "Material" }

func defOp(name string, validate compiler.Validator) compiler.Op {
	return compiler.DefOp("matc."+name, validate)
}

// TODO: ideally we should should follow how MaterialX arranges material
// definition, and MDL to some degree. MDL has a pretty clean way to defining
// materials so we should get inspired by that thing. Actually, MtlX can be
// compiled down to MDL, so maybe we can just model things after MDL?

var (
	OpLoadParameter  = defOp("LoadAttribute", nil)
	OpLoadAttribute2 = defOp("LoadAttribute", nil)
)

// TODO: MaterialX geomprop ops can be either integer or float -typed so we'll
// want to be capable of pulling integer attrs probably. But then I guess we
// also need all the integer ops as well...

// TODO: should take numerical index of the parameter (attribute descriptor)
// rather than a string name.
// TODO: rename or kill (for now)
func LoadAttribute(b *compiler.Builder, attr string) *compiler.Class {
	return b.Value2(OpLoadParameter, core.ArrayType{2, core.Int32}, attr)
}

// param is parameter index
//
// TODO: make this take domain enumeration (and/or mask in the future) so that
// it's used for both loading scene/frame, object and geometry attributes.
// TODO: this should return vec4
func LoadAttribute2(b *compiler.Builder, param int64) *compiler.Class {
	return b.Value2(OpLoadAttribute2, core.ArrayType{2, core.Int32}, param)
}

// TODO: deprecated in favor of LoadAttribute with domain parameter
func LoadObjectProperty(b *compiler.Builder, prop string) *compiler.Class {
	return b.Value2(opInterpreterLoadObjectProperty, core.Int32, prop)
}

var (
	// What should the imm of this be? String filename or a param struct field?
	//
	// String imm has a negative in that we would have to run the full
	// compilation process before we can discover that we already have this
	// material program. On the other hand I guess this is hardly different from
	// materials having various immediates. So I guess let's go with filename
	// string and everything being stuffed into imm? The lowered SampleImage2D
	// would then load up the descriptor or whatever.
	OpSampleImage2D = defOp("SampleImage2D", nil)
)

var (
	// OpGGX = defOp("GGX", nil)

	// OpGeneralizedSchlick = defOp("GeneralizedSchlick", nil)

	// OpNullBSDF = defOp("NullBSDF", nil)

	OpDiffuseBSDF = defOp("DiffuseBSDF", nil)

	// TODO: if we keep one huge MicrofacetBSDF we'll also want to introduce
	// Fresnel type that we'll pass values of to this op.
	OpMicrofacetBSDF = defOp("MicrofacetBSDF", nil)

	// OpNullEDF = defOp("NullEDF", nil)

	// TODO: decide whether to call this uniform (mtlx, osl) or diffuse (mdl)
	OpUniformEDF = defOp("UniformEDF", nil)

	OpBSDFReflectance = defOp("BSDFReflectance", nil) // BSDF -> color

	// Even args are tints (weights), odd args are DFs (TODO: swap?)
	// TODO: rename to something else, e.g. DFCompose or DFWeightedSum? MDL
	// actually has similar functions that it calls mix, so I guess we could
	// also call it DFMix.
	OpDFComposition = defOp("DFComposition", nil)

	OpMakeSurface = defOp("MakeSurface", nil) // (BSDF, EDF) -> Surface

	// OpMakeMaterial = defOp("MakeMaterial", nil) // (Surface, Surface, Volume)

	// TODO: we need an extra "return" instruction with NoReturnType{}
)

var (
	OpDFAdd = defOp("DFAdd", nil) // T : DF, (T, T) -> T

	// TODO: swap parameters so that tint is first and df is second
	OpDFTint = defOp("DFTint", nil) // T : DF, (Vec[3, Float], T) -> T
)

// Rules for flattening BSDF composition and stuff
// var legalizationRules =
