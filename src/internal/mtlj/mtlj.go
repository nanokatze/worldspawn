package mtlj

import (
	"fmt"
	"strconv"

	"github.com/go-json-experiment/json"

	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
	"worldspawn/internal/renderer/matc"
)

type stmt struct {
	Bind string
	Op   string
	Type string
	Imm  string
	Args []string
}

func cooktype(typ string) compiler.Type {
	switch typ {
	case "Empty":
		return core.EmptyType{}
	case "Int[32]":
		return core.Int32
	case "Array[2, Int[32]]":
		return core.ArrayType{2, core.Int32}
	case "Array[3, Int[32]]":
		return core.ArrayType{3, core.Int32}
	case "BSDF":
		return matc.BSDFType{}
	case "EDF":
		return matc.EDFType{}
	default:
		panic("can't handle type " + typ)
	}
}

var opByName = map[string]compiler.Op{
	"IConst":           core.OpIConst,
	"MakeArray":        core.OpMakeArray,
	"ArrayExtract":     core.OpArrayExtract,
	"LoadArgument":     matc.OpLoadArgument,
	"GetShadingNormal": matc.OpInterpreterGetShadingNormal,
	"DiffuseBSDF":      matc.OpDiffuseBSDF,
	"UniformEDF":       matc.OpUniformEDF,
	"DFComposition":    matc.OpDFComposition,
	"MakeSurface":      matc.OpMakeSurface,
}

var opImmParser = map[compiler.Op]func(imm string) (any, error){
	core.OpIConst:       func(imm string) (any, error) { return strconv.ParseInt(imm, 10, 64) },
	core.OpArrayExtract: func(imm string) (any, error) { return strconv.ParseInt(imm, 10, 64) },
	matc.OpLoadArgument: func(imm string) (any, error) {
		idx, err := strconv.ParseInt(imm, 10, 64)
		return int32(idx), err
	},
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

		typ := cooktype(p.Type)

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
