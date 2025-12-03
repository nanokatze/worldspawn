package matc

import (
	"worldspawn/internal/compiler"
	"worldspawn/internal/renderer/internal/material"
)

// type regType struct{ N int }

// Adding a new instruction
//
// No instruction should ever take an array-typed operand. Instructions instead
// should take scalar values and register assignment needs to be aware of how to
// assign registers for operands that must be in adjacent registers and
// assembler needs to be able to insert copies before an instruction to get
// things in a suitable location.

// TODO: interpreter ops should probably use our custom VecN types rather than
// standard tuples? We'll still have to deal with memes like certain ops
// returning (mem, data) etc.

// TODO: could we generalize this to machineOp?
type interpOp interface {
	Validate(typ compiler.Type, imm any, args ...*compiler.Class)
	Assemble(as *assembler, c *compiler.Class, v *compiler.Value, regm map[*compiler.Class]regRange)
}

// TODO: rename to make it clear that it's asm map for interpreter ops
var amap = make(map[compiler.Op]func(as *assembler, class *compiler.Class, v *compiler.Value, regm map[*compiler.Class]regRange))

func defInterpOp(name string, a interpOp) compiler.Op {
	op := defOp("Interp"+name, a.Validate)
	amap[op] = a.Assemble
	return op
}

// TODO: rename
type aaa struct {
	special bool // TODO: kill this and replace with special implementations of interpreterOp
	op      material.A
	dst     bool // TODO: rename to something else?
	arity   int  // TODO: replace with bitmap of args
	imm     bool
}

var (
	opInterpConst32 = defInterpOp("Const32", aaa{op: material.AConst32, dst: true, imm: true})

	opInterpFAddE8M23 = defInterpOp("FAddE8M23", aaa{op: material.AFAddE8M23, dst: true, arity: 2})
	opInterpFSubE8M23 = defInterpOp("FSubE8M23", aaa{op: material.AFSubE8M23, dst: true, arity: 2})
	opInterpFMulE8M23 = defInterpOp("FMulE8M23", aaa{op: material.AFMulE8M23, dst: true, arity: 2})
	opInterpFDivE8M23 = defInterpOp("FDivE8M23", aaa{op: material.AFDivE8M23, dst: true, arity: 2})
	opInterpFMinE8M23 = defInterpOp("FMinE8M23", aaa{op: material.AFMinE8M23, dst: true, arity: 2})
	opInterpFMaxE8M23 = defInterpOp("FMaxE8M23", aaa{op: material.AFMaxE8M23, dst: true, arity: 2})

	opInterpFFloorE8M23 = defInterpOp("FFloorE8M23", aaa{op: material.AFFloorE8M23, dst: true, arity: 1})

	// blender materials actually don't have LessOrEqual, they only have less.
	OpInterpFLessOrEqualE8M23 = defInterpOp("FLessOrEqualE8M23", aaa{op: material.AFLessOrEqualE8M23, dst: true, arity: 2})

	opInterpCondSelect32 = defInterpOp("CondSelect32", aaa{op: material.ACondSelect32, dst: true, arity: 3})

	opInterpLoadArgument  = defInterpOp("LoadArgument", aparam(material.ALoadParam))
	opInterpLoadAttribute = defInterpOp("LoadAttribute", aparam(material.ALoadAttr))

	// TODO: kill in favor of LoadAttribute
	OpInterpLoadShadingNormal = defInterpOp("GetShadingNormal", aaa{op: material.ALoadNormal, dst: true})
)

type aparam material.A

func (d aparam) Validate(typ compiler.Type, imm any, args ...*compiler.Class) {
	_ = imm.(int64)
}

func (d aparam) Assemble(as *assembler, c *compiler.Class, v *compiler.Value, regm map[*compiler.Class]regRange) {
	dst := uint32(regm[c].I)
	off := as.paramOffsets[v.Imm().(int64)]

	as.code = append(as.code, packinstr(material.A(d), dst, 0, 0), uint32(off))
}

// TODO: should be hashable. Doesn't need to be public.
type InterpreterABI struct {
	// TODO: BSDF and EDF are per-surface (and there can be two: front face and
	// back face). Beside that we also need VDFs and also AOVs.
	BSDFOff int
	BSDFs   []material.BSDF
	EDFOff  int
	EDFs    []material.EDF
}

// TODO: these should use their own interpreterOp implementations
var (
	opInterpPseudoArrayExtract = defInterpOp("PseudoArrayExtract", aaa{special: true})

	// TODO: rename this to PseudoMakeMaterial? Or PseudoOutputMaterial. Idk.
	// Must be noreturn.
	OpInterpPseudoOutput = defInterpOp("PseudoOutput", aaa{special: true})
)

func (d aaa) Validate(typ compiler.Type, imm any, args ...*compiler.Class) {
	if d.special {
		return
	}

	if len(args) != d.arity {
		panic("wut")
	}
	if (imm != nil) != d.imm {
		panic("wut 2")
	}
}

func (d aaa) Assemble(as *assembler, class *compiler.Class, v *compiler.Value, regm map[*compiler.Class]regRange) {
	switch v.Op() {
	default:
		if d.special {
			panic("special op")
		}

		a := d.op

		var dst uint32
		if d.dst {
			dst = uint32(regm[class].I)
		}

		srcs := make([]uint32, max(d.arity, 2)) // eww
		for i := range d.arity {
			srcs[i] = uint32(regm[v.Arg(i)].I)
		}

		as.code = append(as.code, packinstr(a, dst, srcs[0], srcs[1]))
		as.code = append(as.code, srcs[2:]...)
		if d.imm {
			as.code = append(as.code, v.Imm().(uint32))
		}

	case OpInterpPseudoOutput:
		// TODO: implement this as parallel copy. Right now, regassign
		// assigns registers in a way that there are no conflicts.

		dst := regm[class].I
		for i, a := range v.Args() {
			// TODO: just check that we don't need to do a parallel copy instead.
			if regm[a].I != dst+i {
				as.code = append(as.code, packinstr(material.ACopy32, uint32(dst+i), uint32(regm[a].I), 0))
			}
		}

	case opInterpPseudoArrayExtract:
		if !d.special {
			panic("must be special")
		}

		if uint32(regm[class].I) != uint32(regm[v.Arg(0)].I)+v.Imm().(uint32) {
			// TODO: do certain assertions and validation here

			as.code = append(as.code, packinstr(material.ACopy32,
				uint32(regm[class].I),
				uint32(regm[v.Arg(0)].I)+v.Imm().(uint32),
				0))
		}
	}
}
