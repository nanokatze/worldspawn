package material

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"worldspawn/internal/renderer/internal/compiler"
)

// TODO: I guess rename to MVM to MatVM? Or MATVM

type assembler struct {
	code []uint32
}

// TODO: separate stuff that lowers IR for interpreter and spits out interpreter
// instruction, and definitions of those instructions assembling

/*
type FloatType struct {
	e int
	m int
}

func (f *FloatType) String() string {
	return fmt.Sprintf("Float[%d, %d]", f.e, f.m)
}

var FloatE8M23 = &FloatType{8, 23}
*/

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
func defMVMOp(name string, a aaa) compiler.Op {
	op := compiler.DefOp(name, nil)
	amap[op] = a
	return op
}

// TODO: actually gen these from somewhere, it's incredibly tedious to rename these
// TODO: move to interpreter.go
var (
	opMVMCopy32 = defMVMOp("MVMCopy32", aaa{special: true})

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

	OpMVMFLessOrEqualE8M23 = defMVMOp("MVMFLessOrEqualE8M23",
		aaa{a: AFLessOrEqualE8M23, dst: true, arity: 2})

	OpMVMConditionalSelect32 = defMVMOp("MVMConditionalSelect32",
		aaa{a: AConditionalSelect32, dst: true, arity: 3})

	OpMVMLoad = defMVMOp("MVMLoad",
		aaa{a: ALoad, dst: true, imm: true})

	OpMVMLoadAttribute = defMVMOp("MVMLoadAttribute",
		aaa{a: ALoadAttribute, dst: true, imm: true})

	OpMVMLoadNormal = defMVMOp("MVMLoadNormal",
		aaa{a: ALoadNormal, dst: true, imm: false})

	opMVMPseudoMakeArray = defMVMOp("MVMPseudoMakeArray",
		aaa{special: true})
	opMVMPseudoArrayExtract = defMVMOp("MVMPseudoArrayExtract",
		aaa{special: true})
)

func shrimpleLowering(arity int, match compiler.Op, _32 compiler.Op) compiler.RewriteRule {
	return compiler.RewriteRule{
		Pattern: &compiler.Pattern{
			Op:   match,
			Args: slices.Repeat([]*compiler.Pattern{{}}, arity),
		},
		Replace: func(sea *compiler.Sea, v *compiler.Value) *compiler.Value {
			bits := v.Type().(compiler.BitsType).N
			switch bits {
			case 32:
				return sea.Value(_32, v.Type(), nil, v.Args...)
			default:
				// TODO: panic with an error object instead
				panic(fmt.Sprintf("cannot lower %d-bit %s", bits, v.Op()))
			}
		},
	}
}

// TODO: make private and apply this at CompileMVMProgram.
var LowerToMVM = []compiler.RewriteRule{
	{
		Pattern: &compiler.Pattern{
			Op:      OpMakeArray,
			ArgsDDD: true,
		},
		Replace: func(sea *compiler.Sea, v *compiler.Value) *compiler.Value {
			return sea.Value(opMVMPseudoMakeArray, v.Type(), nil, v.Args...)
		},
	},
	{
		Pattern: &compiler.Pattern{
			Op: OpArrayExtract,
			Args: []*compiler.Pattern{
				{},
			},
		},
		Replace: func(sea *compiler.Sea, v *compiler.Value) *compiler.Value {
			return sea.Value(opMVMPseudoArrayExtract, v.Type(), uint32(v.Imm().(int64)), v.Args...)
		},
	},
	{
		Pattern: &compiler.Pattern{
			Op: OpConst,
		},
		Replace: func(sea *compiler.Sea, v *compiler.Value) *compiler.Value {
			bits := v.Type().(compiler.BitsType).N
			imm := v.Imm().(int64) // TODO: switch const to immutable bigint
			switch bits {
			case 32:
				return sea.Value(opMVMConst32, v.Type(), uint32(imm))
			default:
				// TODO: panic with an error object instead
				panic(fmt.Sprintf("cannot lower %d-bit %s", bits, v.Op()))
			}
		},
	},
	shrimpleLowering(2, OpFSub, opMVMFSubE8M23),
	shrimpleLowering(2, OpFMin, opMVMFMinE8M23),
	shrimpleLowering(1, OpFFloor, opMVMFFloorE8M23),
}

// TODO: make extract produce an extracted program rather than modify the eq
// classes
func extract(sea *compiler.Sea, c *compiler.Class) {
	for v := range c.Values {
		if _, ok := amap[v.Op()]; !ok {
			sea.KillValue(v)
			continue
		}

		for _, a := range v.Args {
			extract(sea, a)
		}
	}

	// Assert that there's just one insn now.
	_ = c.Value()
}

// TODO: rename to CompileToMatVM? Or have a single compile entry point, with
// different targets being specified by an enum or string or whatever.
func CompileMVMProgram(sea *compiler.Sea, c *compiler.Class) ([]uint32, int) {
	// TODO: lower v with LowerToMVM. We'd probably want to make a copy
	// of v? Or push doing the copy onto the user.

	// log.Println("input")
	// compiler.Dump(sea, v)

	extract(sea, c)

	movins := movInserter{
		sea:     sea,
		visited: make(map[*compiler.Class]struct{}),
		needMov: make(map[*compiler.Class]struct{}),
	}
	movins.do(c)

	compiler.Dump(sea, c, nil)

	sched := schedule2(sea, c)

	regm := regassign2(sea, sched)

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
		for _, a := range v.Args {
			fmt.Fprintf(&sb, " %v", regm[a])
		}
		log.Print(sb.String())
	}

	assembled := assemble(sea, sched, regm)
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
		case AConst32, ALoad, ALoadAttribute:
			data := assembled[i]
			i++
			fmt.Fprintf(os.Stderr, " 0x%08x", data)
		}
		fmt.Fprintln(os.Stderr)
	}

	return assembled, regm[c].I
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

		case opMVMCopy32:
			_ = v.Type().(compiler.BitsType)

			// TODO: make this a part of asm emitter
			if regm[class].I != regm[v.Args[0]].I {
				instrs = append(instrs, packinstr(ACopy32, uint32(regm[class].I), uint32(regm[v.Args[0]].I), 0))
			}

		case opMVMPseudoMakeArray:
			// TODO: implement this as parallel copy. Right now, regassign
			// assigns registers in a way that there are no conflicts.

			dst := regm[class].I
			for i, a := range v.Args {
				instrs = append(instrs, packinstr(ACopy32, uint32(dst+i), uint32(regm[a].I), 0))
			}

		case opMVMPseudoArrayExtract:
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

func regs(t compiler.Type) int {
	if arr, ok := t.(compiler.ArrayType); ok {
		return regs(arr.Elem()) * int(arr.Len())
	}

	bits := t.(compiler.BitsType)
	if bits.N != 32 {
		panic("wtf")
	}
	return 1
}

type scheduler struct {
	sea       *compiler.Sea
	schedule  []*compiler.Class
	scheduled map[*compiler.Class]struct{}
}

func (s *scheduler) allUsesScheduled(c *compiler.Class) bool {
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
	if !s.allUsesScheduled(c) {
		return
	}

	s.schedule = append(s.schedule, c)
	s.scheduled[c] = struct{}{}

	for v := range c.Values2() {
		for _, a := range v.Args {
			s.do(a)
		}
	}
}

func schedule2(sea *compiler.Sea, v *compiler.Class) []*compiler.Class {
	sched := &scheduler{
		sea:       sea,
		schedule:  []*compiler.Class{},
		scheduled: make(map[*compiler.Class]struct{}),
	}
	sched.do(v)

	slices.Reverse(sched.schedule)

	return sched.schedule
}

type extracted struct {
	hmm map[*compiler.Class]*compiler.Value // TODO: key by IDs for perf?
}

// TODO: this should operate on an extracted program
type movInserter struct {
	sea     *compiler.Sea
	movctr  int
	visited map[*compiler.Class]struct{} // TODO:
	needMov map[*compiler.Class]struct{}
}

func (movins *movInserter) do(c *compiler.Class) {
	if _, ok := movins.visited[c]; ok {
		return
	}
	movins.visited[c] = struct{}{}

	v := c.Value()

	// TODO: handle PseudoTupleExtract too?

	if v.Op() == opMVMPseudoMakeArray {
		args := slices.Clone(v.Args)
		for i, a := range args {
			args[i] = movins.useInMakeTuple(a)
		}

		w := movins.sea.Value(opMVMPseudoMakeArray, v.Type(), nil, args...)

		movins.sea.EquateValue(c, w)
		movins.sea.KillValue(v)
	}

	for _, a := range v.Args {
		movins.do(a)
	}
}

func (movins *movInserter) useInMakeTuple(c *compiler.Class) *compiler.Class {
	if _, ok := movins.needMov[c]; ok || true {
		movins.movctr++
		return (&Builder{Sea: movins.sea}).Value2(opMVMCopy32, c.Type(), movins.movctr, c)
	}
	movins.needMov[c] = struct{}{}
	return c
}

type bitset struct {
	words []uint64
}

func (bs bitset) Test(i int) bool {
	return bs.words[i/64]&(1<<(i%64)) != 0
}

func (bs bitset) Set(i int) {
	bs.words[i/64] |= 1 << (i % 64)
}

func (bs bitset) Unset(i int) {
	mask := uint64(1 << (i % 64))
	bs.words[i/64] &^= mask
}

func (bs bitset) FindAndSetMany(n int) int {
outer:
	for i := range 64*len(bs.words) - n {
		for j := range n {
			if bs.Test(i + j) {
				continue outer
			}
		}
		for j := range n {
			bs.Set(i + j)
		}
		return i
	}
	return -1
}

func (bs bitset) UnsetMany(i, n int) {
	for j := range n {
		bs.Unset(i + j)
	}
}

func regassign2(sea *compiler.Sea, sched []*compiler.Class) map[*compiler.Class]regRange {
	killed := make(map[*compiler.Class]int)
	for i, c := range sched {
		for _, a := range c.Value().Args {
			killed[a] = i
		}
	}

	rm := make(map[*compiler.Class]regRange) // TODO: could be keyed by ID for more perf
	bs := bitset{make([]uint64, 4)}
	for i, c := range sched {
		// We don't have parallel copies implemented at the moment, so avoid
		// them for now.
		parallelcopy := c.Value().Op() == opMVMPseudoMakeArray
		if !parallelcopy {
			for k, j := range killed {
				if i == j {
					assignment := rm[k]
					bs.UnsetMany(assignment.I, assignment.N)
				}
			}
		}
		if _, ok := rm[c]; !ok {
			n := regs(c.Type())
			rm[c] = regRange{bs.FindAndSetMany(n), n}
		}
		if parallelcopy {
			for k, j := range killed {
				if i == j {
					assignment := rm[k]
					bs.UnsetMany(assignment.I, assignment.N)
				}
			}
		}
	}

	return rm
}

// TODO: this regassign should operate "backwards" and when we see an
// instruction, we assign registers to its operands immediately. I guess we'll
// need to perform two passes, one will be to establish constraints, and once to
// actually assign stuff I guess? When we assign any instruction that has
// constraints, we'll allocate stuff for the entire constraint.
func regassign3(sea *compiler.Sea, sched []*compiler.Class) map[*compiler.Class]regRange {
	return nil
}
