package matc

import (
	"math"
	"testing"

	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
)

var allRules = append(append([]compiler.RewriteRule(nil), core.Rules...), InterpreterLowerings...)

func TestXxx(t *testing.T) {
	sea := compiler.NewSea()

	b := &compiler.Builder{
		Sea:   sea,
		Rules: allRules,
	}

	paramTypes := []compiler.Type{
		AttributeDescriptorType{}, // uv
		AttributeDescriptorType{}, // color
	}

	paramStruct := MakeParamsTuple(paramTypes)

	normal := b.Value2(OpInterpLoadShadingNormal, core.ArrayType{3, core.Int32}, nil)

	uv := LoadAttribute(b, LoadParameter(b, AttributeDescriptorType{}, 0))
	u := core.ArrayExtract(b, uv, 0)
	v := core.ArrayExtract(b, uv, 1)

	one := core.IConst(b, core.Int32, int64(math.Float32bits(1)))

	fractU := u
	fractV := v

	oneMinusFractU := buildArith2(b, core.OpFSub, one, fractU)
	oneMinusFractV := buildArith2(b, core.OpFSub, one, fractV)

	idk1 := buildArith2(b, core.OpFMin, fractU, fractV)
	idk2 := buildArith2(b, core.OpFMin, oneMinusFractU, oneMinusFractV)
	idk3 := buildArith2(b, core.OpFMin, idk1, idk2)

	selector := b.Value2(
		OpInterpFLessOrEqualE8M23,
		core.Int32,
		nil,
		idk3,
		core.IConst(b, core.Int32, int64(math.Float32bits(0.025))))

	color := LoadAttribute(b, LoadParameter(b, AttributeDescriptorType{}, 1))

	white := core.MakeArray(b, core.Int32, one, one, one, one)

	final := core.CondSelect(b, white, color, selector)

	program := b.Value2(
		OpMakeSurface,
		core.NothingType{},
		nil,
		// bsdf
		b.Value2(OpDFWeightedSum, BSDFType{}, nil,
			final, b.Value2(OpDiffuseBSDF, BSDFType{}, nil, normal)),
		// edf
		b.Value2(OpDFWeightedSum, EDFType{}, nil),
	)

	compiled := CompileInterpretedMaterial(paramStruct, sea, program, nil)

	t.Log(compiled.ABI)
}

func TestXxx2(t *testing.T) {
	sea := compiler.NewSea()

	b := &compiler.Builder{
		Sea:   sea,
		Rules: allRules,
	}

	emission_r := core.IConst(b, core.Int32, int64(math.Float32bits(10.0)))
	emission_g := core.IConst(b, core.Int32, int64(math.Float32bits(10.0)))
	emission_b := core.IConst(b, core.Int32, int64(math.Float32bits(10.0)))
	emissionSpectrum := core.MakeArray(b, core.Int32, emission_r, emission_g, emission_b)
	program := b.Value2(
		OpMakeSurface,
		core.NothingType{},
		nil,
		// bsdf,
		b.Value2(OpDFWeightedSum, BSDFType{}, nil),
		// edf
		b.Value2(OpDFWeightedSum, EDFType{}, nil,
			emissionSpectrum, b.Value2(OpUniformEDF, EDFType{}, nil)),
	)

	compiler.Dump(sea, program, nil)

	// _ = surface
}
