package mc

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"worldspawn/internal/renderer/internal/compiler"
	"worldspawn/internal/renderer/internal/compiler/core"
)

// TODO: I guess rename to MVM to MatVM? Or MATVM

type assembler struct {
	code []uint32
}

// TODO: separate stuff that lowers IR for interpreter and spits out interpreter
// instruction, and definitions of those instructions assembling

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
// TODO: make aaa an interface so we don't need to cram everything into the same
// function. We wanna make it an interface so that we can also implement
// validation this way
func defMVMOp(name string, a aaa) compiler.Op {
	op := compiler.DefOp(name, nil)
	amap[op] = a
	return op
}

var (
	opMVMConst32 = defMVMOp("MVMConst32",
		aaa{a: AConst32, dst: true, imm: true})

	opMVMFAddE8M23 = defMVMOp("MVMFAddE8M23",
		aaa{a: AFAddE8M23, dst: true, arity: 2})
	opMVMFSubE8M23 = defMVMOp("MVMFSubE8M23",
		aaa{a: AFSubE8M23, dst: true, arity: 2})
	opMVMFMulE8M23 = defMVMOp("MVMFMulE8M23",
		aaa{a: AFMulE8M23, dst: true, arity: 2})
	opMVMFDivE8M23 = defMVMOp("MVMFDivE8M23",
		aaa{a: AFDivE8M23, dst: true, arity: 2})
	opMVMFMinE8M23 = defMVMOp("MVMFMinE8M23",
		aaa{a: AFMinE8M23, dst: true, arity: 2})
	opMVMFMaxE8M23 = defMVMOp("MVMFMaxE8M23",
		aaa{a: AFMaxE8M23, dst: true, arity: 2})

	opMVMFFloorE8M23 = defMVMOp("MVMFFloorE8M23",
		aaa{a: AFFloorE8M23, dst: true, arity: 1})

	// blender materials actually don't have LessOrEqual, they only have less.
	OpMVMFLessOrEqualE8M23 = defMVMOp("MVMFLessOrEqualE8M23",
		aaa{a: AFLessOrEqualE8M23, dst: true, arity: 2})

	opMVMCondSelect32 = defMVMOp("MVMCondSelect32",
		aaa{a: ACondSelect32, dst: true, arity: 3})

	// TODO: rename these s/Load/Get/? We usually only use Load for stuff that
	// takes rmem.

	OpMVMLoadParam = defMVMOp("MVMLoadParam",
		aaa{a: ALoadParam, dst: true, imm: true})

	OpMVMLoadAttr = defMVMOp("MVMLoadAttr",
		aaa{a: ALoadAttr, dst: true, imm: true})

	OpMVMLoadNormal = defMVMOp("MVMLoadNormal",
		aaa{a: ALoadNormal, dst: true})

	// TODO: we'll want an instruction per BSDF probably...
	// OpMVMBSDFAlbedo

	// TODO: rename to MakeVec?
	opMVMPseudoMakeArray = defMVMOp("MVMPseudoMakeArray",
		aaa{special: true})
	opMVMPseudoArrayExtract = defMVMOp("MVMPseudoArrayExtract",
		aaa{special: true})
)

func lower(arity int, match compiler.Op, _32 compiler.Op) compiler.RewriteRule {
	return compiler.RewriteRule{
		Pattern: &compiler.Pattern{
			Op:   match,
			Args: slices.Repeat([]*compiler.Pattern{{}}, arity),
		},
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

func lowerCmp(match compiler.Op, _32 compiler.Op) compiler.RewriteRule {
	return compiler.RewriteRule{
		Pattern: &compiler.Pattern{
			Op:   match,
			Args: slices.Repeat([]*compiler.Pattern{{}}, 2),
		},
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

// TODO: kill
var CommonRules = core.Rules

// TODO: make private
var LowerToMVM = []compiler.RewriteRule{
	{
		Pattern: &compiler.Pattern{
			Op:      OpMakeArray,
			ArgsDDD: true,
		},
		Rewrite: func(rc *compiler.RewriteContext, v *compiler.Value) {
			rc.Add2(opMVMPseudoMakeArray, v.Type(), nil, v.Args()...)
		},
	},
	{
		Pattern: &compiler.Pattern{
			Op:   OpArrayExtract,
			Args: []*compiler.Pattern{{}},
		},
		Rewrite: func(rc *compiler.RewriteContext, v *compiler.Value) {
			rc.Add2(opMVMPseudoArrayExtract, v.Type(), uint32(v.Imm().(int64)), v.Args()...)
		},
	},
	{
		Pattern: &compiler.Pattern{
			Op: OpConst,
		},
		Rewrite: func(rc *compiler.RewriteContext, v *compiler.Value) {
			bits := v.Type().(core.IntType).N
			imm := v.Imm().(int64) // TODO: switch const to immutable bigint
			switch bits {
			case 32:
				rc.Add2(opMVMConst32, v.Type(), uint32(imm))
			default:
				// TODO: panic with an error object instead
				panic(fmt.Sprintf("cannot lower %d-bit %s", bits, v.Op()))
			}
		},
	},
	lower(2, OpFSub, opMVMFSubE8M23),
	lower(2, OpFMin, opMVMFMinE8M23),
	lower(1, OpFFloor, opMVMFFloorE8M23),
	lowerCmp(OpFLessOrEqual, OpMVMFLessOrEqualE8M23),

	// TODO: we'll make OpCondSelect's cond 1-bit, while MVMCondSelect32's is
	// 32-bit, so we'll need to consider that when lowering
	{
		Name: "Lower CondSelect",
		Pattern: &compiler.Pattern{
			Op:   OpCondSelect,
			Args: []*compiler.Pattern{{}, {}, {}},
		},
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

func extract2(b *compiler.Rewriter, c *compiler.Class, extracted map[*compiler.Class]*compiler.Class) *compiler.Class {
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

// TODO: rename to CompileToMatVM? Or have a single compile entry point, with
// different targets being specified by an enum or string or whatever.
func CompileMVMProgram(sea *compiler.Sea, c *compiler.Class) ([]uint32, int) {
	// TODO: lower v with LowerToMVM. We'd probably want to make a copy
	// of v? Or push doing the copy onto the user.

	log.Println("input")
	compiler.Dump(sea, c, nil)

	sea2 := compiler.NewSea()

	x := extract2(&compiler.Rewriter{Sea: sea2}, c, make(map[*compiler.Class]*compiler.Class))

	log.Println("extraction")
	compiler.Dump(sea2, x, nil)

	sched := schedule2(x)

	regm := regassign3(sched)

	/*
		compiler.Dump(sea, c, func(c *compiler.Class) string {
			return regm[c].String()
			// return fmt.Sprintf("%v@%v", c.ID, regm[c])
		})
	*/

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

		fmt.Fprintf(os.Stderr, "%v r%v r%v r%v", A(op), dst, src0, src1)
		switch A(op) {
		case AConst32, ALoadParam, ALoadAttr:
			data := assembled[i]
			i++
			fmt.Fprintf(os.Stderr, " 0x%08x", data)
		case ACondSelect32:
			r := assembled[i]
			i++
			fmt.Fprintf(os.Stderr, " r%v", r)
		}
		fmt.Fprintln(os.Stderr)
	}

	return assembled, regm[x].I
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

			instrs = append(instrs, packinstr(a, dst, srcs[0], srcs[1]))
			instrs = append(instrs, srcs[2:]...)
			if amap[v.Op()].imm {
				instrs = append(instrs, v.Imm().(uint32))
			}

		case opMVMPseudoMakeArray:
			// TODO: implement this as parallel copy. Right now, regassign
			// assigns registers in a way that there are no conflicts.

			dst := regm[class].I
			for i, a := range v.Args() {
				// TODO: just check that we don't need to do a parallel copy instead.
				if regm[a].I != dst+i {
					instrs = append(instrs, packinstr(ACopy32, uint32(dst+i), uint32(regm[a].I), 0))
				}
			}

		case opMVMPseudoArrayExtract:
			if !amap[v.Op()].special {
				panic("must be special")
			}

			if uint32(regm[class].I) != uint32(regm[v.Arg(0)].I)+v.Imm().(uint32) {
				// TODO: do certain assertions and validation here

				instrs = append(instrs, packinstr(ACopy32,
					uint32(regm[class].I),
					uint32(regm[v.Arg(0)].I)+v.Imm().(uint32),
					0))
			}
		}
	}

	instrs = append(instrs, packinstr(AStop, 0, 0, 0))

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
