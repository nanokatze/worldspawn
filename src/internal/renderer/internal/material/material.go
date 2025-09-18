package material

import (
	"fmt"
	"log"
	"strings"

	"worldspawn/internal/renderer/internal/compiler"
)

type Builder = compiler.Builder
type Op = compiler.Op
type Value = compiler.Value

// TODO: prefix these differently
const (
	OpStop = iota

	OpMovk32

	OpLoadNormal = 15
)

// TODO: move some ops, types, etc into the base compiler package

func packinstr(op, dst, src0, src1 int) uint32 {
	return uint32(op) | uint32(dst)<<8 | uint32(src0)<<16 | uint32(src1)<<24
}

// TODO: probs gen these from a json or something
// TODO: move these base ops into the compiler package
var (
	OpConst = &Op{Name: "Const"}

	OpMakeTuple        = &Op{Name: "MakeTuple"}
	OpExtractFromTuple = &Op{Name: "ExtractFromTuple"}

	// TODO: change F to be suffix?
	OpFAdd = &Op{Name: "FAdd"}
	OpFSub = &Op{Name: "FSub"}
	OpFMul = &Op{Name: "FMul"}
	OpFDiv = &Op{Name: "FDiv"}
	OpFMin = &Op{Name: "FMin"}
	OpFMax = &Op{Name: "FMax"}

	OpFFloor = &Op{Name: "FFloor"}
)

var (
	// OpSampleTexture = &Op{Name: "SampleTexture"}

	OpXDFAdd   = &Op{Name: "XDFAdd"}
	OpXDFScale = &Op{Name: "XDFScale"}
)

var (
	// TODO: make EDF/BSDF the suffix?
	OpEDFUniform  = &Op{Name: "EDFUniform"} // DiffuseEDF?
	OpBSDFDiffuse = &Op{Name: "BSDFDiffuse"}
)

var (
	OpInterpreterMovk32 = &Op{Name: "InterpreterMovk32"}

	OpInterpreterLoadNormal = &Op{Name: "InterpreterLoadNormal"}

	OpInterpreterPseudoMakeTuple        = &Op{Name: "InterpreterPseudoMakeTuple"}
	OpInterpreterPseudoExtractFromTuple = &Op{Name: "InterpreterPseudoExtractFromTuple"}
)

type valueR struct {
	reg0 int
	regs int
}

type regalloc struct {
	scheduleReverse []*Value
	// scheduled map[*Value]struct{}

	busy uint64 // TODO: should be infinite
	regm map[*Value]valueR
}

func findUnsetBit(x uint64) int {
	for i := range 64 {
		if x&(1<<i) == 0 {
			return i
		}
	}
	panic("oops")
	return -1
}

func (ra *regalloc) do(v *Value) {
	if v.Op != OpInterpreterPseudoMakeTuple {
		ra.scheduleReverse = append(ra.scheduleReverse, v)
	}

	if v.Op == OpInterpreterPseudoMakeTuple {
		r := ra.regm[v]
		i := 0
		for _, a := range v.Args {
			regs := 1
			ra.regm[a] = valueR{r.reg0 + i, regs}
			i += regs
		}
	}

	/*
		if _, ok := ra.regm[v]; !ok {
			r := findUnsetBit(ra.busy)
			ra.busy |= 1 << r
			ra.regm[v] = valueR{r, 1}
		}
	*/

	for _, a := range v.Args {
		ra.do(a)
	}
}

func assemble(scheduleReverse []*Value, regm map[*Value]valueR) []uint32 {
	instrs := []uint32{}

	for i := len(scheduleReverse) - 1; i >= 0; i-- {
		v := scheduleReverse[i]
		r := regm[v]

		switch v.Op {
		case OpInterpreterMovk32:
			instrs = append(instrs, packinstr(OpMovk32, r.reg0, 0, 0), v.Aux.(uint32))
		case OpInterpreterLoadNormal:
			instrs = append(instrs, packinstr(OpLoadNormal, r.reg0, 0, 0))

		default:
			panic("can't assemble op")
		}
	}

	instrs = append(instrs, packinstr(OpStop, 0, 0, 0))

	return instrs
}

func CompileInterpreterProgram(v *Value) []uint32 {
	ra := &regalloc{
		regm: make(map[*Value]valueR),
	}
	ra.busy = 0b111
	ra.regm[v] = valueR{0, 3}
	ra.do(v)

	for i := len(ra.scheduleReverse) - 1; i >= 0; i-- {
		v := ra.scheduleReverse[i]
		r := ra.regm[v]

		var s strings.Builder
		fmt.Fprintf(&s, "r[%d:%d]", r.reg0, r.regs)
		fmt.Fprintf(&s, " = %s", v.Op.Name)
		for _, a := range v.Args {
			r := ra.regm[a]
			fmt.Fprintf(&s, " r[%d:%d]", r.reg0, r.regs)
		}
		if v.Aux != nil {
			fmt.Fprintf(&s, " %v", v.Aux)
		}

		log.Print(s.String())
	}

	return assemble(ra.scheduleReverse, ra.regm)
}
