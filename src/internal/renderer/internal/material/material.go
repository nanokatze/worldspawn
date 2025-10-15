package material

import (
	"fmt"

	"worldspawn/internal/renderer/internal/compiler"
)

// type BxDFType struct{}

/*
type FloatType struct{ E, M int8 }

func (t FloatType) String() string { return fmt.Sprintf("Float[%d, %d]", t.E, t.M) }

func (t FloatType) BitsType() BitsType {
	return MakeBitsType(1 + t.E + t.M)
}
*/

// TODO: replace with a specialized material builder
// TODO: new idea: hide compiler.Op, compiler.Value, and material Builder should
// expose a set of blessed ops on its own. We can also move that builder to the
// base package.

type Builder = compiler.Builder

var (
	OpConst = compiler.OpConst

	OpMakeTuple    = compiler.OpMakeTuple
	OpTupleExtract = compiler.OpTupleExtract

	// TODO: introduce float type which will specify e, m
	OpFAdd = compiler.DefOp("FAdd", nil)
	OpFSub = compiler.DefOp("FSub", nil)
	OpFMul = compiler.DefOp("FMul", nil)
	OpFDiv = compiler.DefOp("FDiv", nil)
	OpFMin = compiler.DefOp("FMin", nil)
	OpFMax = compiler.DefOp("FMax", nil)

	OpFFloor = compiler.DefOp("FFloor", nil)

	OpFEqual       = compiler.DefOp("FEqual", nil)
	OpFLessOrEqual = compiler.DefOp("FLessOrEqual", nil)

	OpCondSelect = compiler.DefOp("CondSelect", nil)
)

/*
var (
	// OpSampleTexture = compiler.DefOp("SampleTexture")

	OpXDFAdd   = compiler.DefOp("XDFAdd")
	OpXDFScale = compiler.DefOp("XDFScale")
)
*/

// TODO: I guess we could also make a universal OpBSDF, but that is kinda sucky
var (
// TODO: make EDF/BSDF the suffix?
// OpEDFUniform  = compiler.DefOp("EDFUniform")
// OpBSDFDiffuse = compiler.DefOp("BSDFDiffuse")
)

// TODO: interpreter ops should probably use our custom VecN types rather than
// standard tuples? We'll still have to deal with memes like certain ops
// returning (mem, data) etc.

type aaa struct {
	special bool // TODO: redo this into variants (a + dst etc, nop, special)
	a       A
	dst     bool
	arity   int // TODO: replace with bitmap of args
	imm     bool
}

var amap = make(map[compiler.Op]aaa)

// TODO: validation
func defInterpreterOp(name string, a aaa) compiler.Op {
	op := compiler.DefOp(name, nil)
	amap[op] = a
	return op
}

// type floatFormat struct{ e, m int8 }

// TODO: actually gen these from somewhere, it's incredibly tedious to rename these
// TODO: move to interpreter.go
var (
	opInterpreterCopy32 = defInterpreterOp("InterpreterCopy32", aaa{special: true})

	opInterpreterConst32 = defInterpreterOp("InterpreterConst32",
		aaa{a: AConst32, dst: true, imm: true})

	opInterpreterFAddE8M23 = defInterpreterOp("InterpreterFAddE8M23",
		aaa{a: AFAddE8M23, dst: true, arity: 2})
	opInterpreterFSubE8M23 = defInterpreterOp("InterpreterFSubE8M23",
		aaa{a: AFSubE8M23, dst: true, arity: 2})
	opInterpreterFMulE8M23 = defInterpreterOp("InterpreterFMulE8M23",
		aaa{a: AFMulE8M23, dst: true, arity: 2})
	opInterpreterFDivE8M23 = defInterpreterOp("InterpreterFDivE8M23",
		aaa{a: AFDivE8M23, dst: true, arity: 2})
	opInterpreterFMinE8M23 = defInterpreterOp("InterpreterFMinE8M23",
		aaa{a: AFMinE8M23, dst: true, arity: 2})
	opInterpreterFMaxE8M23 = defInterpreterOp("InterpreterFMaxE8M23",
		aaa{a: AFMaxE8M23, dst: true, arity: 2})

	opInterpreterFFloorE8M23 = defInterpreterOp("InterpreterFFloorE8M23",
		aaa{a: AFFloorE8M23, dst: true, arity: 1})

	OpInterpreterFLessOrEqualE8M23 = defInterpreterOp("InterpreterFLessOrEqualE8M23",
		aaa{a: AFLessOrEqualE8M23, dst: true, arity: 2})

	OpInterpreterConditionalSelect32 = defInterpreterOp("InterpreterConditionalSelect32",
		aaa{a: AConditionalSelect32, dst: true, arity: 3})

	OpInterpreterLoad = defInterpreterOp("InterpreterLoad",
		aaa{a: ALoad, dst: true, imm: true})

	OpInterpreterLoadAttribute = defInterpreterOp("InterpreterLoadAttribute",
		aaa{a: ALoadAttribute, dst: true, imm: true})

	OpInterpreterLoadNormal = defInterpreterOp("InterpreterLoadNormal",
		aaa{a: ALoadNormal, dst: true, imm: false})

	opInterpreterPseudoMakeTuple = defInterpreterOp("InterpreterPseudoMakeTuple",
		aaa{special: true})
	opInterpreterPseudoTupleExtract = defInterpreterOp("InterpreterPseudoTupleExtract",
		aaa{special: true})
)

type regRange struct{ I, N int }

func (rr regRange) String() string {
	// if rr.n < 1 {
	// 	panic("wat")
	// }
	if rr.N == 1 {
		return fmt.Sprintf("r%d", rr.I)
	}
	return fmt.Sprintf("r[%d:%d]", rr.I, rr.I+rr.N-1)
}

func assemble(sea *compiler.Sea, schedule []*compiler.Class, regm map[*compiler.Class]regRange) []uint32 {
	instrs := []uint32{}

	for _, class := range schedule {
		v := class.Value()

		if _, ok := amap[v.Op()]; !ok {
			panic("can't assemble op " + v.Op().String())
		}

		switch v.Op() {
		default:
			if amap[v.Op()].special {
				panic("special op")
			}

			a := amap[v.Op()].a

			var dst uint32
			if amap[v.Op()].dst {
				dst = uint32(regm[class].I)
			}

			srcs := make([]uint32, max(amap[v.Op()].arity, 2)) // eww
			for i := range amap[v.Op()].arity {
				srcs[i] = uint32(regm[v.Args[i]].I)
			}

			instrs = append(instrs, packinstr(a, dst, srcs[0], srcs[1]))
			instrs = append(instrs, srcs[2:]...)
			if amap[v.Op()].imm {
				instrs = append(instrs, v.Imm().(uint32))
			}

		case opInterpreterCopy32:
			_ = v.Type().(compiler.BitsType)

			instrs = append(instrs, packinstr(ACopy32, uint32(regm[class].I), uint32(regm[v.Args[0]].I), 0))

		case opInterpreterPseudoMakeTuple:
			// It's basically a pcopy
			// panic("implement")

		case opInterpreterPseudoTupleExtract:
			if !amap[v.Op()].special {
				panic("must be special")
			}

			// TODO: do certain assertions and validation here

			instrs = append(instrs, packinstr(ACopy32,
				uint32(regm[class].I),
				uint32(regm[v.Args[0]].I)+v.Imm().(uint32),
				0))
		}
	}

	instrs = append(instrs, packinstr(AStop, 0, 0, 0))

	return instrs
}
