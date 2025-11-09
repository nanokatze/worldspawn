package matc

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
	"worldspawn/internal/renderer/internal/material"
)

func cost(v *compiler.Value) int {
	if v.Op() == opInterpreterConst32 {
		return 0
	}
	if v.Op() == opInterpreterPseudoArrayExtract {
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

type InterpretedMaterial struct {
	Code         []uint32
	OutputLayout InterpretedMaterialOutputLayout
	Outputs      int
}

func CompileForInterpreter(sea *compiler.Sea, c *compiler.Class) *InterpretedMaterial {
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

	itable := x.Value().Imm().(*InterpretedMaterialOutputLayout)

	// itable.BSDFOff

	return &InterpretedMaterial{
		Code:         assembled,
		OutputLayout: *itable,
		Outputs:      regm[x].I,
	}
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
