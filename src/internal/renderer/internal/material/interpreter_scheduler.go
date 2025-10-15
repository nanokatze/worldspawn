package material

import (
	"slices"

	"worldspawn/internal/renderer/internal/compiler"
)

func regs(t compiler.Type) int {
	if tup, ok := t.(*compiler.TupleType); ok {
		n := 0
		for _, e := range tup.Elems() {
			n += regs(e)
		}
		return n
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

	if v.Op() == opInterpreterPseudoMakeTuple {
		args := slices.Clone(v.Args)
		for i, a := range args {
			args[i] = movins.useInMakeTuple(a)
		}

		w := movins.sea.Value(opInterpreterPseudoMakeTuple, v.Type(), nil, args...)

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
		return (&Builder{Sea: movins.sea}).Value2(opInterpreterCopy32, c.Type(), movins.movctr, c)
	}
	movins.needMov[c] = struct{}{}
	return c
}

// TODO: kill in favor of regassign2
func regassign(sea *compiler.Sea, sched []*compiler.Class) map[*compiler.Class]regRange {
	rm := make(map[*compiler.Class]regRange) // TODO: could be keyed by ID for more perf
	regnext := 0                             // TODO: replace with an actual allocator
	for _, c := range slices.Backward(sched) {
		if _, ok := rm[c]; !ok {
			n := regs(c.Type())
			rm[c] = regRange{regnext, n}
			regnext += n
		}

		v := c.Value()
		if v.Op() == opInterpreterPseudoMakeTuple {
			i := rm[c].I
			for _, a := range v.Args {
				// TODO: assert that a doesn't have regm assigned
				if _, ok := rm[a]; ok {
					panic("conflict")
				}
				n := regs(a.Type())
				rm[a] = regRange{i, n}
				i += n
			}
		}
	}

	return rm
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
		for k, j := range killed {
			if i == j {
				assignment := rm[k]
				bs.UnsetMany(assignment.I, assignment.N)
			}
		}
		if _, ok := rm[c]; !ok {
			n := regs(c.Type())
			rm[c] = regRange{bs.FindAndSetMany(n), n}
		}
	}

	return rm
}
