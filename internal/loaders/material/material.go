package material

import (
	"fmt"
	"io/fs"
	"strconv"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"

	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
	"worldspawn/internal/renderer/matc"
)

// TODO: move wmaterial-specific code into a subdir

// TODO: do not depend on matc here. Only compiler/material at most.

// TODO: the material object returned by Load should only contain a program with
// the compiler/material instructions only. It should not contain renderer
// preamble and renderer shader, nor any of the renderer/matc instructions.

type header struct {
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

// TODO: this should not be here at all, but rather in the compiler's IR parser
func Type(typ string) compiler.Type {
	switch typ {
	case "Nothing":
		return core.NothingType{}
	case "AttributeDescriptor":
		return matc.AttributeDescriptorType{}
	case "Bits[32]":
		return core.Bits32
	case "Bits[64]":
		return core.Bits64
	case "Bits[96]":
		return core.Bits96
	case "Bits[128]":
		return core.Bits128
	case "BSDF":
		return matc.BSDFType{}
	case "EDF":
		return matc.EDFType{}
	default:
		panic("can't handle type " + typ)
	}
}

var opByName = map[string]compiler.Op{
	"Const":             core.OpConst,
	"DFWeightedSum":     matc.OpDFWeightedSum,
	"DiffuseBSDF":       matc.OpDiffuseBSDF,
	"Extract":           core.OpExtract,
	"LoadAttribute":     matc.OpLoadAttribute,
	"LoadParameter":     matc.OpLoadParameter,
	"LoadShadingNormal": matc.OpInterpLoadShadingNormal,
	"MakeSurface":       matc.OpMakeSurface,
	"Pack":              core.OpPack,
	"UniformEDF":        matc.OpUniformEDF,
}

var opImmParser = map[compiler.Op]func(imm string) (any, error){
	core.OpConst:         func(imm string) (any, error) { return strconv.ParseUint(imm, 10, 64) },
	core.OpExtract:       func(imm string) (any, error) { return strconv.ParseInt(imm, 10, 64) },
	matc.OpLoadParameter: func(imm string) (any, error) { return strconv.ParseInt(imm, 10, 64) },
}

// TODO: proper syntax
func parse(b *compiler.Builder, src []byte) (*compiler.Class, error) {
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

// TODO: this would be moved to the common package material loader
// TODO: can we get rid of this and return just *compiler.Class? Or even the
// still-serialized blob that will have to be parsed, but by the compiler's
// parser.
type Material struct {
	// TODO: we should change renderer's material api to accept a blob of code
	// in compiler's material dialect.

	Params   []string
	Preamble []string
	IR       *compiler.Class
}

// TODO: allow user to pass the compiler rules somehow?
func Load(fsys fs.FS, filename string) (*Material, error) {
	src, err := fs.ReadFile(fsys, filename)
	if err != nil {
		return nil, err
	}

	var header header
	if err := json.Unmarshal(src, &header); err != nil {
		return nil, err
	}

	sea := compiler.NewSea()
	b := &compiler.Builder{
		Sea: sea,
		// TODO: don't use any rules at this point. We should just decode the
		// program. Or I guess the user could optionally supply the rules.
		Rules: append(append([]compiler.RewriteRule(nil), core.Rules...), matc.LowerToInterpreter...),
	}
	ir, err := parse(b, header.Program)
	if err != nil {
		return nil, err
	}

	return &Material{
		Params:   header.Params,
		Preamble: header.Preamble,
		IR:       ir,
	}, nil
}
