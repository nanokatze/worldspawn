package matc

import (
	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
	"worldspawn/internal/renderer/internal/material"
)

// TODO: plop interpreter optimization rules into a separate rule set?

var InterpreterLowerings = []compiler.RewriteRule{
	// TODO: we'll make OpCondSelect's cond 1-bit, while InterpCondSelect32's is
	// 32-bit, so we'll need to consider that when lowering
	{
		Comment: "interpreter lowering",
		Pattern: &compiler.Pattern{Op: core.OpCondSelect, Args: []*compiler.Pattern{{}, {}, {}}},
		Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
			if v.Type() != core.Int32 {
				return
			}
			rr.Add2(opInterpCondSelect32, v.Type(), nil, v.Args()...)
		},
	},

	{
		Comment: "interpreter lowering",
		Pattern: &compiler.Pattern{Op: core.OpArrayExtract, Args: []*compiler.Pattern{{}}},
		Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
			rr.Add2(opInterpPseudoArrayExtract, v.Type(), uint32(v.Imm().(int64)), v.Args()...)
		},
	},

	{
		Comment: "interpreter lowering",
		Pattern: &compiler.Pattern{Op: core.OpIConst},
		Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
			if v.Type() != core.Int32 {
				return
			}
			rr.Add2(opInterpConst32, v.Type(), uint32(v.Imm().(int64)))
		},
	},

	interpLowerArith(core.OpFAdd, opInterpFAddE8M23),
	interpLowerArith(core.OpFSub, opInterpFSubE8M23),
	interpLowerArith(core.OpFMul, opInterpFMulE8M23),
	interpLowerArith(core.OpFDiv, opInterpFDivE8M23),
	interpLowerArith(core.OpFMin, opInterpFMinE8M23),
	interpLowerArith(core.OpFMax, opInterpFMaxE8M23),
	interpLowerArith(core.OpFFloor, opInterpFFloorE8M23),
	interpLowerCmp(core.OpFLessOrEqual, OpInterpFLessOrEqualE8M23),

	{
		Comment: "interpreter lowering",
		Pattern: &compiler.Pattern{Op: OpLoadMaterialParameter},
		Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
			if _, ok := v.Type().(core.IntType); !ok {
				return
			}
			rr.Add2(opInterpLoadArgument, v.Type(), v.Imm())
		},
	},
	{
		Comment: "interpreter lowering",
		Pattern: &compiler.Pattern{Op: OpLoadAttribute, Args: []*compiler.Pattern{{Op: OpLoadMaterialParameter}}},
		Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
			_ = v.Arg(0).Type().(AttributeDescriptor)

			for w := range v.Arg(0).Values() {
				if w.Op() == OpLoadMaterialParameter {
					rr.Add2(opInterpLoadAttribute, v.Type(), w.Imm().(int32))
				}
			}
		},
	},

	{
		Comment: "interpreter lowering",
		Pattern: &compiler.Pattern{Op: OpMakeSurface, Args: []*compiler.Pattern{{}, {}}},
		Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
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

			rr.Add2(OpInterpPseudoOutput, core.EmptyType{}, &abi, args...)
		},
	},
}

func interpLowerArith(match compiler.Op, _32 compiler.Op) compiler.RewriteRule {
	return compiler.RewriteRule{
		Comment: "interpreter lowering",
		Pattern: &compiler.Pattern{Op: match, ArgsDDD: true},
		Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
			if v.Type() != core.Int32 {
				return
			}
			rr.Add2(_32, v.Type(), nil, v.Args()...)
		},
	}
}

func interpLowerCmp(match compiler.Op, _32 compiler.Op) compiler.RewriteRule {
	return compiler.RewriteRule{
		Comment: "interpreter lowering",
		Pattern: &compiler.Pattern{Op: match, ArgsDDD: true},
		Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
			if v.Arg(0).Type() != core.Int32 {
				return
			}
			// TODO: std comparisons return 1-bit values, while we return
			// Bits32. So we need a helper op to bridge this gap.
			rr.Add2(_32, core.Int32, nil, v.Args()...)
		},
	}
}
