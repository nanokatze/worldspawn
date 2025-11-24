package matc

import (
	"fmt"

	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
	"worldspawn/internal/renderer/internal/material"
)

// TODO: plop interpreter optimization rules into a separate rule set?

var LowerToInterpreter = []compiler.RewriteRule{
	// TODO: we'll make OpCondSelect's cond 1-bit, while
	// InterpreterCondSelect32's is 32-bit, so we'll need to consider that when
	// lowering
	{
		Name:    "Lower CondSelect",
		Pattern: &compiler.Pattern{Op: core.OpCondSelect, Args: []*compiler.Pattern{{}, {}, {}}},
		Rewrite: func(b *compiler.Builder, r *compiler.RewriteResult, v *compiler.Value) {
			bits, ok := v.Type().(core.IntType)
			if !ok {
				return
			}
			switch bits.N {
			case 32:
				r.Add2(opInterpreterCondSelect32, v.Type(), nil, v.Args()...)
			default:
				// TODO: panic with an error object instead
				panic(fmt.Sprintf("cannot lower %d-bit %s", bits, v.Op()))
			}
		},
	},

	{
		Name:    "Lower ArrayExtract",
		Pattern: &compiler.Pattern{Op: core.OpArrayExtract, Args: []*compiler.Pattern{{}}},
		Rewrite: func(b *compiler.Builder, r *compiler.RewriteResult, v *compiler.Value) {
			r.Add2(opInterpreterPseudoArrayExtract, v.Type(), uint32(v.Imm().(int64)), v.Args()...)
		},
	},

	{
		Name:    "Lower IConst",
		Pattern: &compiler.Pattern{Op: core.OpIConst},
		Rewrite: func(b *compiler.Builder, r *compiler.RewriteResult, v *compiler.Value) {
			bits := v.Type().(core.IntType).N
			imm := v.Imm().(int64)
			switch bits {
			case 32:
				r.Add2(opInterpreterConst32, v.Type(), uint32(imm))
			default:
				// TODO: panic with an error object instead
				panic(fmt.Sprintf("cannot lower %d-bit %s", bits, v.Op()))
			}
		},
	},

	lowerFloatArith(core.OpFAdd, opInterpreterFAddE8M23),
	lowerFloatArith(core.OpFSub, opInterpreterFSubE8M23),
	lowerFloatArith(core.OpFMul, opInterpreterFMulE8M23),
	lowerFloatArith(core.OpFDiv, opInterpreterFDivE8M23),
	lowerFloatArith(core.OpFMin, opInterpreterFMinE8M23),
	lowerFloatArith(core.OpFMax, opInterpreterFMaxE8M23),
	lowerFloatArith(core.OpFFloor, opInterpreterFFloorE8M23),
	lowerFloatCmp(core.OpFLessOrEqual, OpInterpreterFLessOrEqualE8M23),

	{
		Name:    fmt.Sprintf("Lower %s (interpreter)", OpLoadParameter),
		Pattern: &compiler.Pattern{Op: OpLoadParameter},
		Rewrite: func(b *compiler.Builder, r *compiler.RewriteResult, v *compiler.Value) {
			r.Add2(opInterpreterLoadAttribute, v.Type(), v.Imm(), v.Args()...)
		},
	},

	{
		Name:    "Lower MakeSurface",
		Pattern: &compiler.Pattern{Op: OpMakeSurface, Args: []*compiler.Pattern{{}, {}}},
		Rewrite: func(b *compiler.Builder, r *compiler.RewriteResult, v *compiler.Value) {
			// TODO: don't assume that there's just a single instruction
			bsdf := v.Arg(0).Value()
			edf := v.Arg(1).Value()

			var abi InterpreterABI
			var args, args2 []*compiler.Class

			// TODO: factor common parts out somehow pls

			abi.BSDFOff = len(args)
			for i := range len(bsdf.Args()) / 2 {
				tint := bsdf.Arg(2*i + 0)
				tintR := core.ArrayExtract(b, tint, 0)
				tintG := core.ArrayExtract(b, tint, 1)
				tintB := core.ArrayExtract(b, tint, 2)
				args = append(args, tintR, tintG, tintB)

				// TODO: don't assume there's just a single instruction in the
				// class. Expand stuff correctly.
				df := bsdf.Arg(2*i + 1).Value()

				switch df.Op() {
				case OpDiffuseBSDF:
					abi.BSDFs = append(abi.BSDFs, material.BSDFDiffuse)
					n := df.Arg(0)
					nX := core.ArrayExtract(b, n, 0)
					nY := core.ArrayExtract(b, n, 1)
					nZ := core.ArrayExtract(b, n, 2)
					args2 = append(args2, nX, nY, nZ)

				default:
					panic("bad")
				}
			}
			args = append(args, args2...)
			args2 = args[:0]

			abi.EDFOff = len(args)
			for i := range len(edf.Args()) / 2 {
				tint := edf.Arg(2*i + 0)
				tintR := core.ArrayExtract(b, tint, 0)
				tintG := core.ArrayExtract(b, tint, 1)
				tintB := core.ArrayExtract(b, tint, 2)
				args = append(args, tintR, tintG, tintB)

				// TODO: don't assume there's just a single instruction in the
				// class. Expand stuff correctly.
				df := edf.Arg(2*i + 1).Value()

				switch df.Op() {
				case OpUniformEDF:
					abi.EDFs = append(abi.EDFs, material.EDFUniform)

				default:
					panic("bad")
				}
			}
			args = append(args, args2...)
			args2 = args[:0]

			r.Add2(OpInterpreterPseudoOutput, core.EmptyType{}, &abi, args...)
		},
	},
}

func lowerFloatArith(match compiler.Op, _32 compiler.Op) compiler.RewriteRule {
	return compiler.RewriteRule{
		Pattern: &compiler.Pattern{Op: match, ArgsDDD: true},
		Rewrite: func(b *compiler.Builder, r *compiler.RewriteResult, v *compiler.Value) {
			bits := v.Type().(core.IntType).N
			switch bits {
			case 32:
				r.Add2(_32, v.Type(), nil, v.Args()...)
			default:
				// TODO: panic with an error object instead
				panic(fmt.Sprintf("cannot lower %d-bit %s", bits, v.Op()))
			}
		},
	}
}

func lowerFloatCmp(match compiler.Op, _32 compiler.Op) compiler.RewriteRule {
	return compiler.RewriteRule{
		Pattern: &compiler.Pattern{Op: match, ArgsDDD: true},
		Rewrite: func(b *compiler.Builder, r *compiler.RewriteResult, v *compiler.Value) {
			bits := v.Type().(core.IntType).N
			switch bits {
			case 32:
				// TODO: std comparisons return 1-bit values, while we return
				// Bits32. So we need a helper op to bridge this gap.
				r.Add2(_32, core.Int32, nil, v.Args()...)
			default:
				// TODO: panic with an error object instead
				panic(fmt.Sprintf("cannot lower %d-bit %s", bits, v.Op()))
			}
		},
	}
}
