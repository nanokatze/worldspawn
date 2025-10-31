package mc

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	"worldspawn/internal/renderer/internal/compiler"
	"worldspawn/internal/renderer/internal/compiler/core"
	"worldspawn/internal/renderer/internal/material"
)

// TODO: should assembler live here or in material? I feel like here is pretty
// nice, but then we should move packinstr back here.
type assembler struct {
	code []uint32
}

// TODO: interpreter ops should probably use our custom VecN types rather than
// standard tuples? We'll still have to deal with memes like certain ops
// returning (mem, data) etc.

// TODO: make aaa an interface so we don't need to cram everything into the same
// function. We wanna make it an interface so that we can also implement
// validation this way
type aaa struct {
	special bool
	a       material.A
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

// Adding a new instruction
//
// No instruction should ever take an array-typed operand. Instructions instead
// should take scalar values and register assignment needs to be aware of how to
// assign registers for operands that must be in adjacent registers and
// assembler needs to be able to insert copies before an instruction to get
// things in a suitable location.
var (
	opMVMConst32 = defInterpreterOp("MVMConst32",
		aaa{a: material.AConst32, dst: true, imm: true})

	opMVMFAddE8M23 = defInterpreterOp("MVMFAddE8M23",
		aaa{a: material.AFAddE8M23, dst: true, arity: 2})
	opMVMFSubE8M23 = defInterpreterOp("MVMFSubE8M23",
		aaa{a: material.AFSubE8M23, dst: true, arity: 2})
	opMVMFMulE8M23 = defInterpreterOp("MVMFMulE8M23",
		aaa{a: material.AFMulE8M23, dst: true, arity: 2})
	opMVMFDivE8M23 = defInterpreterOp("MVMFDivE8M23",
		aaa{a: material.AFDivE8M23, dst: true, arity: 2})
	opMVMFMinE8M23 = defInterpreterOp("MVMFMinE8M23",
		aaa{a: material.AFMinE8M23, dst: true, arity: 2})
	opMVMFMaxE8M23 = defInterpreterOp("MVMFMaxE8M23",
		aaa{a: material.AFMaxE8M23, dst: true, arity: 2})

	opMVMFFloorE8M23 = defInterpreterOp("MVMFFloorE8M23",
		aaa{a: material.AFFloorE8M23, dst: true, arity: 1})

	// blender materials actually don't have LessOrEqual, they only have less.
	OpMVMFLessOrEqualE8M23 = defInterpreterOp("MVMFLessOrEqualE8M23",
		aaa{a: material.AFLessOrEqualE8M23, dst: true, arity: 2})

	opMVMCondSelect32 = defInterpreterOp("MVMCondSelect32",
		aaa{a: material.ACondSelect32, dst: true, arity: 3})

	// TODO: rename these s/Load/Get/? We usually only use Load for stuff that
	// takes rmem.

	OpMVMLoadParam = defInterpreterOp("MVMLoadParam",
		aaa{a: material.ALoadParam, dst: true, imm: true})

	OpMVMLoadAttr = defInterpreterOp("MVMLoadAttr",
		aaa{a: material.ALoadAttr, dst: true, imm: true})

	OpMVMLoadNormal = defInterpreterOp("MVMLoadNormal",
		aaa{a: material.ALoadNormal, dst: true})

	// TODO: we'll want an instruction per BSDF probably...
	// OpMVMBSDFDiffuseAlbedo

	// TODO: rename to something better like OutputValues or idk
	//
	// BSDF tints
	// BSDF parameters
	// EDF tints
	// EDF parameters
	OpMVMPseudoOutput = defInterpreterOp("MVMPseudoOutput",
		aaa{special: true})

	opMVMPseudoArrayExtract = defInterpreterOp("MVMPseudoArrayExtract",
		aaa{special: true})
)

func lowerFloatArith(match compiler.Op, _32 compiler.Op) compiler.RewriteRule {
	return compiler.RewriteRule{
		Pattern: &compiler.Pattern{Op: match, ArgsDDD: true},
		Rewrite: func(rc *compiler.RewriteContext, v *compiler.Value) {
			bits := v.Type().(core.IntType).N
			switch bits {
			case 32:
				rc.Add2(_32, v.Type(), nil, v.Args()...)
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
		Rewrite: func(rc *compiler.RewriteContext, v *compiler.Value) {
			bits := v.Type().(core.IntType).N
			switch bits {
			case 32:
				// TODO: std comparisons return 1-bit values, while we return
				// Bits32. So we need a helper op to bridge this gap.
				rc.Add2(_32, core.Int32, nil, v.Args()...)
			default:
				// TODO: panic with an error object instead
				panic(fmt.Sprintf("cannot lower %d-bit %s", bits, v.Op()))
			}
		},
	}
}

var LowerToInterpreter = []compiler.RewriteRule{
	{
		Name:    "Lower Const",
		Pattern: &compiler.Pattern{Op: OpIConst},
		Rewrite: func(rc *compiler.RewriteContext, v *compiler.Value) {
			bits := v.Type().(core.IntType).N
			imm := v.Imm().(int64)
			switch bits {
			case 32:
				rc.Add2(opMVMConst32, v.Type(), uint32(imm))
			default:
				// TODO: panic with an error object instead
				panic(fmt.Sprintf("cannot lower %d-bit %s", bits, v.Op()))
			}
		},
	},

	{
		Pattern: &compiler.Pattern{Op: OpArrayExtract, Args: []*compiler.Pattern{{}}},
		Rewrite: func(rc *compiler.RewriteContext, v *compiler.Value) {
			rc.Add2(opMVMPseudoArrayExtract, v.Type(), uint32(v.Imm().(int64)), v.Args()...)
		},
	},
	lowerFloatArith(OpFSub, opMVMFSubE8M23),
	lowerFloatArith(OpFMin, opMVMFMinE8M23),
	lowerFloatArith(OpFFloor, opMVMFFloorE8M23),
	lowerFloatCmp(OpFLessOrEqual, OpMVMFLessOrEqualE8M23),

	// TODO: we'll make OpCondSelect's cond 1-bit, while MVMCondSelect32's is
	// 32-bit, so we'll need to consider that when lowering
	{
		Name:    "Lower CondSelect",
		Pattern: &compiler.Pattern{Op: OpCondSelect, Args: []*compiler.Pattern{{}, {}, {}}},
		Rewrite: func(rc *compiler.RewriteContext, v *compiler.Value) {
			bits, ok := v.Type().(core.IntType)
			if !ok {
				return
			}
			switch bits.N {
			case 32:
				rc.Add2(opMVMCondSelect32, v.Type(), nil, v.Args()...)
			default:
				// TODO: panic with an error object instead
				panic(fmt.Sprintf("cannot lower %d-bit %s", bits, v.Op()))
			}
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

func cost(v *compiler.Value) int {
	if v.Op() == opMVMConst32 {
		return 0
	}
	if v.Op() == opMVMPseudoArrayExtract {
		return 2
	}
	return 1
}

func extract2(b *compiler.Builder, c *compiler.Class, extracted map[*compiler.Class]*compiler.Class) *compiler.Class {
	// TODO: this can lead to infinite loops if not done carefully, explain why
	// it's ok here and possibly rewrite extractor to not use Newest().
	c = c.Newest()

	if x, ok := extracted[c]; ok {
		return x
	}

	var best *compiler.Value
	for v := range c.Values() {
		if _, ok := amap[v.Op()]; !ok {
			continue
		}
		if best == nil || cost(best) > cost(v) {
			best = v
		}
	}
	if best == nil {
		panic("whoopsydoopsy")
	}

	args := make([]*compiler.Class, len(best.Args()))
	for i := range args {
		args[i] = extract2(b, best.Arg(i), extracted)
	}

	x := b.Value2(best.Op(), best.Type(), best.Imm(), args...)

	extracted[c] = x

	return x
}

type InterpretedMaterial struct {
	Code         []uint32
	OutputLayout material.InterpretedMaterialOutputLayout
	Outputs      int
}

const (
	TargetInterpreter = 1
	// TargetVulkanSpv
)

// TODO: add a way to identify the target
// TODO: return a container object
func Compile(sea *compiler.Sea, c *compiler.Class, target int) *InterpretedMaterial {
	t0 := time.Now()
	defer func() { log.Println("Compile", time.Since(t0)) }()

	log.Println("input")
	compiler.Dump(sea, c, nil)

	sea2 := compiler.NewSea()

	x := extract2(&compiler.Builder{Sea: sea2}, c, make(map[*compiler.Class]*compiler.Class))

	// log.Println("extraction")
	// compiler.Dump(sea2, x, nil)

	sched := schedule2(x)

	regm := regassign3(sched)

	for _, c := range sched {
		v := c.Value()
		var sb strings.Builder
		fmt.Fprintf(&sb, "%v", regm[c])
		// fmt.Fprintf(&sb, " %v", v.Type)
		fmt.Fprintf(&sb, " = %s", v.Op())
		if imm := v.Imm(); imm != nil {
			fmt.Fprintf(&sb, " %v", imm)
		}
		for _, a := range v.Args() {
			fmt.Fprintf(&sb, " %v", regm[a])
		}
		log.Print(sb.String())
	}

	assembled := assemble(sched, regm)
	log.Println("disassembly")
	for i := 0; i < len(assembled); {
		w := assembled[i]
		i++

		op := w & 0xff
		dst := (w >> 8) & 0xff
		src0 := (w >> 16) & 0xff
		src1 := (w >> 24) & 0xff

		fmt.Fprintf(os.Stderr, "%v r%v r%v r%v", material.A(op), dst, src0, src1)
		switch material.A(op) {
		case material.AConst32, material.ALoadParam, material.ALoadAttr:
			data := assembled[i]
			i++
			fmt.Fprintf(os.Stderr, " 0x%08x", data)
		case material.ACondSelect32:
			r := assembled[i]
			i++
			fmt.Fprintf(os.Stderr, " r%v", r)
		}
		fmt.Fprintln(os.Stderr)
	}

	itable := x.Value().Imm().(*material.InterpretedMaterialOutputLayout)

	// itable.BSDFOff

	return &InterpretedMaterial{
		Code:         assembled,
		OutputLayout: *itable,
		Outputs:      regm[x].I,
	}
}

func assemble(schedule []*compiler.Class, regm map[*compiler.Class]regRange) []uint32 {
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
				srcs[i] = uint32(regm[v.Arg(i)].I)
			}

			instrs = append(instrs, material.Packinstr(a, dst, srcs[0], srcs[1]))
			instrs = append(instrs, srcs[2:]...)
			if amap[v.Op()].imm {
				instrs = append(instrs, v.Imm().(uint32))
			}

		case OpMVMPseudoOutput:
			// TODO: implement this as parallel copy. Right now, regassign
			// assigns registers in a way that there are no conflicts.

			dst := regm[class].I
			for i, a := range v.Args() {
				// TODO: just check that we don't need to do a parallel copy instead.
				if regm[a].I != dst+i {
					instrs = append(instrs, material.Packinstr(material.ACopy32, uint32(dst+i), uint32(regm[a].I), 0))
				}
			}

		case opMVMPseudoArrayExtract:
			if !amap[v.Op()].special {
				panic("must be special")
			}

			if uint32(regm[class].I) != uint32(regm[v.Arg(0)].I)+v.Imm().(uint32) {
				// TODO: do certain assertions and validation here

				instrs = append(instrs, material.Packinstr(material.ACopy32,
					uint32(regm[class].I),
					uint32(regm[v.Arg(0)].I)+v.Imm().(uint32),
					0))
			}
		}
	}

	instrs = append(instrs, material.Packinstr(material.AStop, 0, 0, 0))

	return instrs
}

func regs(t compiler.Type) int {
	if arr, ok := t.(core.ArrayType); ok {
		return regs(arr.Elem()) * int(arr.Len())
	}

	bits := t.(core.IntType)
	if bits.N != 32 {
		panic("wtf")
	}
	return 1
}

// TODO: come up with a nicer way to represent schedule. Needs to be a linked
// list.
// TODO: glue scheduling to extraction?
type scheduler struct {
	schedule  []*compiler.Class
	scheduled map[*compiler.Class]struct{}
}

func (s *scheduler) allUsersScheduled(c *compiler.Class) bool {
	for u := range c.Users() {
		if _, ok := s.scheduled[u.Class()]; !ok {
			return false
		}
	}
	return true
}

func (s *scheduler) do(c *compiler.Class) {
	if _, ok := s.scheduled[c]; ok {
		return
	}
	if !s.allUsersScheduled(c) {
		return
	}

	s.schedule = append(s.schedule, c)
	s.scheduled[c] = struct{}{}

	v := c.Value()

	for _, a := range v.Args() {
		s.do(a)
	}
}

func schedule2(v *compiler.Class) []*compiler.Class {
	sched := &scheduler{
		schedule:  []*compiler.Class{},
		scheduled: make(map[*compiler.Class]struct{}),
	}
	sched.do(v)

	slices.Reverse(sched.schedule)

	return sched.schedule
}
