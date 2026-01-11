package wmaterial

import (
	"fmt"
	"strconv"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"

	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
	"worldspawn/internal/pathtracer/matc"
)

type Header struct {
	// TODO: params probs should be folded into both program and preamble, or
	// alternatively preamble could be made unaware of param types.
	Params   []string
	Preamble []string
	Program  jsontext.Value
}

type stmt struct {
	Bind string
	Op   string
	Type string
	Imm  string
	Args []string
}

func Type(typ string) compiler.Type {
	switch typ {
	case "Nothing":
		return core.NothingType{}
	case "AttributeDescriptor":
		return matc.AttributeDescriptorType{}
	case "Int[32]":
		return core.Bits32
	case "Int[64]", "Array[2, Int[32]]":
		return core.Bits64
	case "Int[96]", "Array[3, Int[32]]":
		return core.BitsType{96}
	case "Int[128]", "Array[4, Int[32]]":
		return core.BitsType{128}
	case "BSDF":
		return matc.BSDFType{}
	case "EDF":
		return matc.EDFType{}
	default:
		panic("can't handle type " + typ)
	}
}

var opByName = map[string]compiler.Op{
	"ArrayExtract":      core.OpExtract,
	"DFWeightedSum":     matc.OpDFWeightedSum,
	"DiffuseBSDF":       matc.OpDiffuseBSDF,
	"IConst":            core.OpConst,
	"LoadAttribute":     matc.OpLoadAttribute,
	"LoadParameter":     matc.OpLoadParameter,
	"LoadShadingNormal": matc.OpInterpLoadShadingNormal,
	"MakeArray":         core.OpPack,
	"MakeSurface":       matc.OpMakeSurface,
	"UniformEDF":        matc.OpUniformEDF,
}

var opImmParser = map[compiler.Op]func(imm string) (any, error){
	core.OpConst: func(imm string) (any, error) { return strconv.ParseUint(imm, 10, 64) },
	core.OpExtract: func(imm string) (any, error) {
		idx, err := strconv.ParseInt(imm, 10, 64)
		return idx * 32, err
	},
	matc.OpLoadParameter: func(imm string) (any, error) { return strconv.ParseInt(imm, 10, 64) },
}

// TODO: a proper syntax?
func Parse(b *compiler.Builder, src []byte) (*compiler.Class, error) {
	var prog []stmt
	if err := json.Unmarshal(src, &prog); err != nil {
		return nil, err
	}

	m := make(map[string]*compiler.Class)
	var l *compiler.Class
	for _, p := range prog {
		op := opByName[p.Op]
		if op == (compiler.Op{}) {
			panic(fmt.Sprintf("cock %s", p.Op))
		}

		typ := Type(p.Type)

		var imm any
		immparser := opImmParser[op]
		if immparser != nil {
			var err error
			imm, err = immparser(p.Imm)
			if err != nil {
				panic(err)
			}
		}

		args := make([]*compiler.Class, len(p.Args))
		for i, a := range p.Args {
			c, ok := m[a]
			if !ok {
				panic("balls")
			}
			args[i] = c
		}

		l = b.Value2(op, typ, imm, args...)
		m[p.Bind] = l
	}

	return l, nil
}
