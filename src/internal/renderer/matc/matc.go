package matc

import (
	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
)

// TODO: split this file?

func defOp(name string, validate compiler.Validator) compiler.Op {
	return compiler.DefOp("matc."+name, validate)
}

var OpLoadMaterialParameter = defOp("LoadParameter", nil)

func LoadMaterialParameter(b *compiler.Builder, typ compiler.Type, index int32) *compiler.Class {
	return b.Value2(OpLoadMaterialParameter, typ, index)
}

// TODO: put other fields into AttributeDescriptor (e.g. the set of domains it
// can be pulled from etc)
type AttributeDescriptor struct{}

func (AttributeDescriptor) String() string { return "AttributeDescriptor" }

// TODO: should we have a version of LoadAttribute that, if attribute is
// per-vertex, returns uninterpolated attribute chosen randomly instead of
// interpolating? This would be useful for basically advanced vertex colors.
var OpLoadAttribute = defOp("LoadAttribute", nil)

// TODO: should be a vec4
func LoadAttribute(b *compiler.Builder, arg *compiler.Class) *compiler.Class {
	return b.Value2(OpLoadAttribute, core.ArrayType{2, core.Int32}, nil, arg)
}

// TODO: unify these into a single type? We have some ops that are defined for
// either BSDF or EDF.
type (
	BSDFType struct{}
	EDFType  struct{}
)

func (BSDFType) String() string { return "BSDF" }
func (EDFType) String() string  { return "EDF" }

type (
	SurfaceType struct{}
	VolumeType  struct{}
)

func (SurfaceType) String() string { return "Surface" }
func (VolumeType) String() string  { return "Volume" }

type MaterialType struct{}

func (MaterialType) String() string { return "Material" }

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

	OpMakeSurface = defOp("MakeSurface", nil) // (BSDF, EDF) -> Surface

	// OpMakeMaterial = defOp("MakeMaterial", nil) // (Surface, Surface, Volume)

	// TODO: we need an extra "return" instruction with NoReturnType{}
)

var (
	OpDFAdd = defOp("DFAdd", nil) // T : DF, (T, T) -> T

	// TODO: swap parameters so that tint is first and df is second
	OpDFTint = defOp("DFTint", nil) // T : DF, (Vec[3, Float], T) -> T

	// Even args are tints (weights), odd args are DFs (TODO: swap?)
	OpDFWeightedSum = defOp("DFWeightedSum", nil)
)

// Rules for flattening BSDF composition and stuff
// var legalizationRules =
