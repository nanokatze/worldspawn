package mc

import (
	"fmt"

	"worldspawn/internal/renderer/internal/compiler"
	"worldspawn/internal/renderer/internal/compiler/core"
)

// TODO: plop interpreter optimization rules into a separate rule set?

var LowerToInterpreter = []compiler.RewriteRule{
	{
		Name:    "Lower IConst",
		Pattern: &compiler.Pattern{Op: core.OpIConst},
		Rewrite: func(b *compiler.Builder, r *compiler.Rewriter, v *compiler.Value) {
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

	// TODO: we'll make OpCondSelect's cond 1-bit, while
	// InterpreterCondSelect32's is 32-bit, so we'll need to consider that when
	// lowering
	{
		Name:    "Lower CondSelect",
		Pattern: &compiler.Pattern{Op: core.OpCondSelect, Args: []*compiler.Pattern{{}, {}, {}}},
		Rewrite: func(b *compiler.Builder, r *compiler.Rewriter, v *compiler.Value) {
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
		Pattern: &compiler.Pattern{Op: core.OpArrayExtract, Args: []*compiler.Pattern{{}}},
		Rewrite: func(b *compiler.Builder, r *compiler.Rewriter, v *compiler.Value) {
			r.Add2(opInterpreterPseudoArrayExtract, v.Type(), uint32(v.Imm().(int64)), v.Args()...)
		},
	},

	/*
		{
			Name:    "Lower DFComposition",
			Pattern: &compiler.Pattern{Op: OpDFComposition, ArgsDDD: true},
			Rewrite: func(rc *compiler.RewriteContext, v *compiler.Value) {
				b := rc.B()

				var args []*compiler.Class

				for i := range len(v.Args()) / 2 {
					tint := v.Arg(2*i + 0)

					tint_r := core.ArrayExtract(b, tint, 0)
					tint_g := core.ArrayExtract(b, tint, 1)
					tint_b := core.ArrayExtract(b, tint, 2)

					args = append(args, tint_r, tint_g, tint_b)
				}

				for i := range len(v.Args()) / 2 {
					df := v.Arg(2*i + 1).Value()

					switch df.Op() {
					case OpDiffuseBSDF:
						n := df.Arg(0)
						n_x := core.ArrayExtract(b, n, 0)
						n_y := core.ArrayExtract(b, n, 1)
						n_z := core.ArrayExtract(b, n, 2)
						args = append(args, n_x, n_y, n_z)

					case OpUniformEDF:

					default:
						panic("oh no")
					}
				}

				rc.Add2(OpMVMPseudoOutput, core.MakeArrayType(int64(len(args)), core.Int32), nil, args...)
			},
		},
	*/
}

func lowerFloatArith(match compiler.Op, _32 compiler.Op) compiler.RewriteRule {
	return compiler.RewriteRule{
		Pattern: &compiler.Pattern{Op: match, ArgsDDD: true},
		Rewrite: func(b *compiler.Builder, r *compiler.Rewriter, v *compiler.Value) {
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
		Rewrite: func(b *compiler.Builder, r *compiler.Rewriter, v *compiler.Value) {
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
