package matc

import (
	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
)

// TODO: ideally we should should follow how MaterialX arranges material
// definition, and MDL to some degree. MDL has a pretty clean way to defining
// materials so we should get inspired by that thing. Actually, MtlX can be
// compiled down to MDL, so maybe we can just model things after MDL?

var (
	// TODO: OpLoadAttr{Scene,Object,Geometry}

	// TODO: rename LoadAttrObject to LoadObjectProperty and LoadAttrGeometry to
	// just LoadAttribute?

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
	// OpGGX = defOp("GGX", nil)

	// OpGeneralizedSchlick = defOp("GeneralizedSchlick", nil)

	OpDiffuseBSDF = defOp("DiffuseBSDF", nil)

	// TODO: if we keep one huge MicrofacetBSDF we'll also want to introduce
	// Fresnel type that we'll pass values of to this op.
	OpMicrofacetBSDF = defOp("MicrofacetBSDF", nil)

	// TODO: decide whether to call this uniform (mtlx, osl) or diffuse (mdl)
	OpUniformEDF = defOp("UniformEDF", nil)

	OpBSDFReflectance = defOp("BSDFReflectance", nil) // BSDF -> color

	OpDFAdd = defOp("DFAdd", nil) // T : DF, (T, T) -> DF

	// TODO: rename to e.g. DFTint?
	OpDFScale = defOp("DFScale", nil) // T : DF, (T, float32) -> T

	// Even args are weights, odd args are DF values (TODO: swap?)
	// TODO: rename to something else
	OpDFComposition = defOp("DFComposition", nil)

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

func defOp(name string, validate compiler.ValidationFunc) compiler.Op {
	return compiler.DefOp("matc."+name, validate)
}
