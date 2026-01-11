package matc

import (
	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
	"worldspawn/internal/pathtracer/internal/material"
)

// TODO: plop interpreter optimization rules into a separate rule set?

var LowerToInterpreter = []compiler.RewriteRule{
	// Idk if this rule belongs here really?
	{
		Comment: "split large CondSelects into many smaller ones",
		Pattern: &compiler.Pattern{Op: core.OpCondSelect, Args: []*compiler.Pattern{{}, {}, {}}},
		Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
			if v.Type().(core.BitsType).N <= 32 {
				return
			}

			x := v.Arg(0)
			y := v.Arg(1)
			cond := v.Arg(2)

			result := make([]*compiler.Class, v.Type().(core.BitsType).N/32)
			for i := int64(0); i < v.Type().(core.BitsType).N; i += 32 {
				result[i/32] = core.CondSelect(b,
					core.Extract(b, core.Bits32, x, i),
					core.Extract(b, core.Bits32, y, i),
					cond)
			}
			rr.Add2(core.OpPack, v.Type(), nil, result...)
		},
	},

	// TODO: we'll make OpCondSelect's cond 1-bit, while InterpCondSelect32's is
	// 32-bit, so we'll need to consider that when lowering
	{
		Comment: "lower to interpreter",
		Pattern: &compiler.Pattern{Op: core.OpCondSelect, Args: []*compiler.Pattern{{}, {}, {}}},
		Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
			if v.Type() == core.Bits32 {
				rr.Add2(opInterpCondSelect32, v.Type(), nil, v.Args()...)
			}
		},
	},

	{
		Comment: "lower to interpreter",
		Pattern: &compiler.Pattern{Op: core.OpExtract, Args: []*compiler.Pattern{{}}},
		Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
			off := v.Imm().(int64)
			if off%32 == 0 {
				rr.Add2(opInterpPseudoArrayExtract, v.Type(), uint32(off/32), v.Args()...)
			}
		},
	},

	{
		Comment: "lower to interpreter",
		Pattern: &compiler.Pattern{Op: core.OpConst},
		Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
			if v.Type() == core.Bits32 {
				rr.Add2(opInterpConst32, v.Type(), uint32(v.Imm().(uint64)))
			}
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
		Comment: "lower to interpreter",
		Pattern: &compiler.Pattern{Op: OpLoadAttribute, Args: []*compiler.Pattern{{Op: OpLoadParameter}}},
		Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
			_ = v.Arg(0).Type().(AttributeDescriptorType)

			for w := range v.Arg(0).Values() {
				if w.Op() == OpLoadParameter {
					rr.Add2(opInterpLoadAttribute, v.Type(), w.Imm().(int64))
				}
			}
		},
	},

	{
		Comment: "lower to interpreter",
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
				tintR := core.Extract(b, core.Bits32, tint, 0)
				tintG := core.Extract(b, core.Bits32, tint, 32)
				tintB := core.Extract(b, core.Bits32, tint, 64)
				args = append(args, tintR, tintG, tintB)

				// TODO: don't assume there's just a single instruction in the
				// class. Expand stuff correctly.
				df := bsdf.Arg(2*i + 1).Value()

				switch df.Op() {
				case OpDiffuseBSDF:
					abi.BSDFs = append(abi.BSDFs, material.BSDFDiffuse)
					n := df.Arg(0)
					nX := core.Extract(b, core.Bits32, n, 0)
					nY := core.Extract(b, core.Bits32, n, 32)
					nZ := core.Extract(b, core.Bits32, n, 64)
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
				tintR := core.Extract(b, core.Bits32, tint, 0)
				tintG := core.Extract(b, core.Bits32, tint, 32)
				tintB := core.Extract(b, core.Bits32, tint, 64)
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

			rr.Add2(OpInterpPseudoOutput, core.NothingType{}, &abi, args...)
		},
	},
}

func interpLowerArith(match compiler.Op, _32 compiler.Op) compiler.RewriteRule {
	return compiler.RewriteRule{
		Comment: "lower to interpreter",
		Pattern: &compiler.Pattern{Op: match, ArgsDDD: true},
		Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
			if v.Type() != core.Bits32 {
				return
			}
			rr.Add2(_32, v.Type(), nil, v.Args()...)
		},
	}
}

func interpLowerCmp(match compiler.Op, _32 compiler.Op) compiler.RewriteRule {
	return compiler.RewriteRule{
		Comment: "lower to interpreter",
		Pattern: &compiler.Pattern{Op: match, ArgsDDD: true},
		Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
			if v.Arg(0).Type() != core.Bits32 {
				return
			}
			// TODO: std comparisons return 1-bit values, while we return
			// Bits32. So we need a helper op to bridge this gap.
			rr.Add2(_32, core.Bits32, nil, v.Args()...)
		},
	}
}
