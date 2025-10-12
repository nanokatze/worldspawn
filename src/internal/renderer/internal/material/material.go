package material

import (
	"fmt"
	"log"
	"strings"

	"worldspawn/internal/renderer/internal/compiler"
)

// TODO: replace with a specialized material builder
// TODO: new idea: hide compiler.Op, compiler.Value, and material Builder should
// expose a set of blessed ops on its own. We can also move that builder to the
// base package.
type Builder = compiler.Builder

type Op = compiler.Op

type Value = compiler.Value

var (
	OpConst = compiler.OpConst

	OpMakeTuple    = compiler.OpMakeTuple
	OpTupleExtract = compiler.OpTupleExtract

	OpFAdd = compiler.DefOp("FAdd")
	OpFSub = compiler.DefOp("FSub")
	OpFMul = compiler.DefOp("FMul")
	OpFDiv = compiler.DefOp("FDiv")
	OpFMin = compiler.DefOp("FMin")
	OpFMax = compiler.DefOp("FMax")

	OpFFloor = compiler.DefOp("FFloor")
)

/*
var (
	// OpSampleTexture = compiler.DefOp("SampleTexture")

	OpXDFAdd   = compiler.DefOp("XDFAdd")
	OpXDFScale = compiler.DefOp("XDFScale")
)

var (
	// TODO: make EDF/BSDF the suffix?
	OpEDFUniform  = compiler.DefOp("EDFUniform" // DiffuseEDF?)
	OpBSDFDiffuse = compiler.DefOp("BSDFDiffuse")
)
*/

// TODO: interpreter ops should probably use our custom VecN types rather than
// standard tuples? We'll still have to deal with memes like certain ops
// returning (mem, data) etc.

type aaa struct {
	special bool // TODO: redo this into variants (a + dst etc, nop, special)
	nop     bool
	a       A
	dst     bool
	arity   int // TODO: replace with bitmap of args
	imm     bool
}

var amap = make(map[compiler.Op]aaa)

func withA(a aaa) func(op compiler.Op) { return func(op Op) { amap[op] = a } }

// TODO: actually gen these from somewhere, it's incredibly tedious to rename these
var (
	OpInterpreterCopy32 = compiler.DefOp("InterpreterCopy32",
		withA(aaa{a: ACopy32, dst: true, arity: 1}))

	OpInterpreterConst32 = compiler.DefOp("InterpreterConst32",
		withA(aaa{a: AConst32, dst: true, imm: true}))

	OpInterpreterFAdd32 = compiler.DefOp("InterpreterFAdd32",
		withA(aaa{a: AFAdd32, dst: true, arity: 2}))
	OpInterpreterFSub32 = compiler.DefOp("InterpreterFSub32",
		withA(aaa{a: AFSub32, dst: true, arity: 2}))
	OpInterpreterFMul32 = compiler.DefOp("InterpreterFMul32",
		withA(aaa{a: AFMul32, dst: true, arity: 2}))
	OpInterpreterFDiv32 = compiler.DefOp("InterpreterFDiv32",
		withA(aaa{a: AFDiv32, dst: true, arity: 2}))
	OpInterpreterFMin32 = compiler.DefOp("InterpreterFMin32",
		withA(aaa{a: AFMin32, dst: true, arity: 2}))
	OpInterpreterFMax32 = compiler.DefOp("InterpreterFMax32",
		withA(aaa{a: AFMax32, dst: true, arity: 2}))

	OpInterpreterFFloor32 = compiler.DefOp("InterpreterFFloor32",
		withA(aaa{a: AFFloor32, dst: true, arity: 1}))

	OpInterpreterFLessOrEqual32 = compiler.DefOp("InterpreterFLessOrEqual32",
		withA(aaa{a: AFLessOrEqual32, dst: true, arity: 2}))

	OpInterpreterConditionalSelect32 = compiler.DefOp("InterpreterConditionalSelect32",
		withA(aaa{a: AConditionalSelect32, dst: true, arity: 3}))

	OpInterpreterLoad = compiler.DefOp("InterpreterLoad",
		withA(aaa{a: ALoad, dst: true, imm: true}))

	OpInterpreterLoadAttribute = compiler.DefOp("InterpreterLoadAttribute",
		withA(aaa{a: ALoadAttribute, dst: true, imm: true}))

	OpInterpreterLoadNormal = compiler.DefOp("InterpreterLoadNormal",
		withA(aaa{a: ALoadNormal, dst: true}))

	OpInterpreterPseudoMakeTuple = compiler.DefOp("InterpreterPseudoMakeTuple",
		withA(aaa{nop: true}))
	OpInterpreterPseudoTupleExtract = compiler.DefOp("InterpreterPseudoTupleExtract",
		withA(aaa{special: true}))
)

type regRange struct{ i, n int }

func (rr regRange) String() string {
	// if rr.n < 1 {
	// 	panic("wat")
	// }
	if rr.n == 1 {
		return fmt.Sprintf("r%d", rr.i)
	}
	return fmt.Sprintf("r[%d:%d]", rr.i, rr.i+rr.n-1)
}

func assemble(schedule []*Value, regm map[*Value]regRange) []uint32 {
	instrs := []uint32{}

	for _, v := range schedule {
		if _, ok := amap[v.Op]; !ok {
			panic("can't assemble op " + v.Op.String())
		}

		switch v.Op {
		default:
			if amap[v.Op].special {
				panic("special op")
			}

			if amap[v.Op].nop {
				// skip assembling pseudo ops
				continue
			}

			a := amap[v.Op].a

			var dst uint32
			if amap[v.Op].dst {
				dst = uint32(regm[v].i)
			}

			srcs := make([]uint32, max(amap[v.Op].arity, 2)) // eww
			for i := range amap[v.Op].arity {
				srcs[i] = uint32(regm[v.Args[i]].i)
			}

			instrs = append(instrs, packinstr(a, dst, srcs[0], srcs[1]))
			instrs = append(instrs, srcs[2:]...)
			if amap[v.Op].imm {
				instrs = append(instrs, v.Imm.(uint32))
			}

		case OpInterpreterPseudoTupleExtract:
			if !amap[v.Op].special {
				panic("must be special")
			}

			// TODO: do certain assertions and validation here

			instrs = append(instrs, packinstr(ACopy32, uint32(regm[v].i), uint32(regm[v.Args[0]].i+v.Imm.(int)), 0))
		}
	}

	instrs = append(instrs, packinstr(AStop, 0, 0, 0))

	return instrs
}

func CompileInterpreterProgram(sea *compiler.Sea, v *Value) []uint32 {
	// This is ass.
	sched, rm := schedule(v)

	for _, v := range sched {
		var sb strings.Builder
		fmt.Fprintf(&sb, "%v", rm[v])
		// fmt.Fprintf(&sb, " %v", v.Type)
		fmt.Fprintf(&sb, " = %s", v.Op)
		for _, a := range v.Args {
			fmt.Fprintf(&sb, " %v", rm[a])
		}
		if v.Imm != nil {
			fmt.Fprintf(&sb, " %v", v.Imm)
		}
		log.Print(sb.String())
	}

	return assemble(sched, rm)
}
