package material

import (
	"log"
	"testing"
)

func TestXxx(t *testing.T) {
	_a := &Value{Op: OpInterpreterMovk32, Aux: uint32(0)}
	_b := &Value{Op: OpInterpreterMovk32, Aux: uint32(0)}
	_c := &Value{Op: OpInterpreterMovk32, Aux: uint32(0)}
	_threeZeros := &Value{Op: OpInterpreterPseudoMakeTuple, Args: []*Value{_a, _b, _c}}

	compiledProgram := CompileInterpreterProgram(_threeZeros)
	for _, x := range compiledProgram {
		log.Printf("0x%08x", x)
	}
}
