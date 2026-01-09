package matc

import (
	"fmt"
	"io"
	"slices"

	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
	"worldspawn/internal/pathtracer/internal/material"
)

func cost(v *compiler.Value) int {
	if v.Op() == opInterpConst32 {
		return 0
	}
	if v.Op() == opInterpPseudoArrayExtract {
		return 2
	}
	return 1
}

// TODO: when we add control flow, extraction will need to be aware of what can
// be scheduled. Also it's possible to express a class that contains no
// instruction that dominates all users, in which case a class would have to be
// duplicated, but it's not clear whether we could realistically end up in such
// situation.
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

// This is basically the same as renderer.InterpretedMaterial
type InterpretedMaterial struct {
	ABI  material.InterpreterABI
	Code []uint32
}

// TODO: AOVs. Either the user can specify aov name -> offset mapping at compile
// time, we can do the remapping later somehow
// TODO: allow printing stuff for debugging somehow
func CompileInterpretedMaterial(params ParamsTuple, sea *compiler.Sea, c *compiler.Class, log io.Writer) *InterpretedMaterial {
	sea2 := compiler.NewSea()

	// TODO: dump representation to log if we have log

	x := extract2(&compiler.Builder{Sea: sea2}, c, make(map[*compiler.Class]*compiler.Class))

	sched := schedule2(x)

	regm := regassign3(sched)

	if log != nil {
		fmt.Fprintln(log, "schedule")
		for _, c := range sched {
			v := c.Value()
			fmt.Fprintf(log, "%v", regm[c])
			fmt.Fprintf(log, " = %s", v.Op())
			if imm := v.Imm(); imm != nil {
				fmt.Fprintf(log, " %v", imm)
			}
			for _, a := range v.Args() {
				fmt.Fprintf(log, " %v", regm[a])
			}
			fmt.Fprintln(log)
		}
	}

	assembled := assemble(sched, regm, params)
	if log != nil {
		fmt.Fprintln(log, "disassembly")
		for i := 0; i < len(assembled); {
			w := assembled[i]
			i++

			op := w & 0xff
			dst := (w >> 8) & 0xff
			src0 := (w >> 16) & 0xff
			src1 := (w >> 24) & 0xff

			fmt.Fprintf(log, "%v r%v r%v r%v", material.A(op), dst, src0, src1)
			switch material.A(op) {
			case material.AConst32, material.ALoadAttribute:
				data := assembled[i]
				i++
				fmt.Fprintf(log, " 0x%08x", data)
			case material.ACondSelect32:
				r := assembled[i]
				i++
				fmt.Fprintf(log, " r%v", r)
			}
			fmt.Fprintln(log)
		}
	}

	abi := x.Value().Imm().(*InterpreterABI)

	// TODO: change material.InterpreterABI to be a string/byte bag and use that
	// throughout

	var bsdfs [4]material.BSDF
	for i, num := range abi.BSDFs {
		bsdfs[i] = num
	}

	var edfs [1]material.EDF
	for i, num := range abi.EDFs {
		edfs[i] = num
	}

	return &InterpretedMaterial{
		ABI: material.InterpreterABI{
			BSDFs:     bsdfs,
			BSDFCount: uint8(len(abi.BSDFs)),
			BSDFsOff:  uint8(abi.BSDFOff),

			EDFs:     edfs,
			EDFCount: uint8(len(abi.EDFs)),
			EDFsOff:  uint8(abi.EDFOff),

			OutputsReg: uint32(regm[x].I),
		},
		Code: assembled,
	}
}

func regs(t compiler.Type) int {
	if _, ok := t.(core.NothingType); ok {
		return 0
	}

	if arr, ok := t.(core.ArrayType); ok {
		return regs(arr.T) * int(arr.N)
	}

	bits := t.(core.IntType)
	if bits.N != 32 {
		panic("wtf")
	}
	return 1
}

// TODO: come up with a nicer way to represent schedule. Needs to be a linked
// list.
type scheduler struct {
	// TODO: when we start representing the extracted program as a map from
	// class to a value in that class, we can kill this again
	uses map[*compiler.Class]map[*compiler.Value]struct{}

	schedule  []*compiler.Class
	scheduled map[*compiler.Class]struct{}
}

func (s *scheduler) allUsersScheduled(c *compiler.Class) bool {
	for u := range s.uses[c] {
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
		uses: make(map[*compiler.Class]map[*compiler.Value]struct{}),

		schedule:  []*compiler.Class{},
		scheduled: make(map[*compiler.Class]struct{}),
	}

	tmp := make(map[*compiler.Class]struct{})
	var f func(c *compiler.Class)
	f = func(c *compiler.Class) {
		if _, ok := tmp[c]; ok {
			return
		}
		tmp[c] = struct{}{}

		for v := range c.Values() {
			for _, a := range v.Args() {
				if sched.uses[a] == nil {
					sched.uses[a] = make(map[*compiler.Value]struct{})
				}
				sched.uses[a][v] = struct{}{}

				f(a)
			}
		}
	}
	f(v)

	sched.do(v)

	slices.Reverse(sched.schedule)

	return sched.schedule
}
