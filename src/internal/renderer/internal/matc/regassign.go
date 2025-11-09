package matc

import (
	"fmt"
	"slices"

	"worldspawn/internal/compiler"
)

// TODO: clean this up

type regRange struct{ I, N int }

// TODO: change notation to something with more consistency between N==1 and N>1
// cases?
func (rr regRange) String() string {
	// if rr.n < 1 {
	// 	panic("wat")
	// }
	if rr.N == 1 {
		return fmt.Sprintf("r%d", rr.I)
	}
	return fmt.Sprintf("r[%d:%d]", rr.I, rr.I+rr.N-1)
}

type constraint struct {
	N        int
	Classes  map[*compiler.Class]int
	assigned bool
}

type regassigner struct {
	constraints map[*compiler.Class]*constraint
	bs          bitset
	m           map[*compiler.Class]regRange
}

func (ra *regassigner) assign(a *compiler.Class) {
	if rr, ok := ra.m[a]; ok {
		for i := range rr.N {
			ra.bs.Set(rr.I + i)
		}
	}

	c := ra.constraints[a]

	if !c.assigned {
		i0 := ra.bs.FindAndSetMany(c.N)

		for k, i := range c.Classes {
			n := regs(k.Type())
			rr := regRange{i0 + i, n}
			if rr2, ok := ra.m[k]; ok && rr2 != rr {
				panic("bad")
			}
			ra.m[k] = rr
		}

		c.assigned = true
	}
}

func (ra *regassigner) free(rr regRange) {
	ra.bs.UnsetMany(rr.I, rr.N)
}

func constrain(m map[*compiler.Class]*constraint, l, r *compiler.Class, d int) {
	k_l := m[l]
	k_r := m[r]

	// There's only two cases possible:
	// c_l != c_r and c_l.Classes and c_r.Classes do not share any classes
	// c_l == c_r

	if k_l != k_r {
		// Plop classes from k_r into k_l, such that k_l.Classes[l] + d =
		// k_r.Classes[r]
		off := k_l.Classes[l] + d - k_r.Classes[r]

		for c, o := range k_r.Classes {
			k_l.Classes[c] = off + o
			m[c] = k_l
		}

		k_l.N = max(k_l.N, off+k_r.N)
	}
	/* can't constrain further otherwise */
}

// TODO: establish these while assigning instead of performing a pass ahead of
// time
func gatherConstraints(sched []*compiler.Class) map[*compiler.Class]*constraint {
	constraints := make(map[*compiler.Class]*constraint)
	for _, c := range sched {
		v := c.Value()

		if constraints[c] == nil {
			constraints[c] = &constraint{
				N:       regs(c.Type()),
				Classes: map[*compiler.Class]int{c: 0},
			}
		}

		switch v.Op() {
		case OpInterpreterPseudoOutput:
			for i, a := range v.Args() {
				constrain(constraints, c, a, i)
			}

		case opInterpreterPseudoArrayExtract:
			constrain(constraints, v.Arg(0), c, int(v.Imm().(uint32)))
		}
	}

	return constraints
}

func regassign3(sched []*compiler.Class) map[*compiler.Class]regRange {
	constraints := gatherConstraints(sched)

	ra := regassigner{
		constraints: constraints,
		bs:          bitset{make([]uint64, 4)},
		m:           make(map[*compiler.Class]regRange),
	}

	ra.assign(sched[len(sched)-1])

	for _, c := range slices.Backward(sched) {
		v := c.Value()

		// TODO: our PseudoOutput special case is busted, we should handle it like
		// everything else and just properly emit parallel copy when assembling.
		pcopy := v.Op() == OpInterpreterPseudoOutput

		if !pcopy {
			ra.free(ra.m[c])
		}

		for _, a := range v.Args() {
			ra.assign(a)
		}

		if pcopy {
			ra.free(ra.m[c])

			for _, a := range v.Args() {
				ra.assign(a)
			}
		}
	}

	return ra.m
}
