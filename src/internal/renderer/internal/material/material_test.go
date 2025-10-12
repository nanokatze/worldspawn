package material

import (
	"log"
	"math"
	"testing"

	"worldspawn/internal/renderer/internal/compiler"
)

func TestXxx(t *testing.T) {
	sea := compiler.NewSea()

	b := Builder{
		Sea: sea,
	}

	_normal := b.Value(OpInterpreterLoadNormal, compiler.MakeTupleType(compiler.Bits32, compiler.Bits32, compiler.Bits32), nil, math.Float32bits(0.0))
	_uv := b.Value(OpInterpreterLoadAttribute, compiler.MakeTupleType(compiler.Bits32, compiler.Bits32), nil, uint32(42))
	_v := b.Value(OpInterpreterPseudoTupleExtract, compiler.Bits32, []*compiler.Value{_uv}, 1)
	_color := compiler.BuildTuple(&b,
		OpInterpreterPseudoMakeTuple,
		_normal,
		_v,
		_v,
		_v,
	)

	compiledProgram := CompileInterpreterProgram(sea, _color)
	for _, x := range compiledProgram {
		log.Printf("0x%08x", x)
	}
}
