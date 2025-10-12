package material

import (
	"slices"
	"worldspawn/internal/renderer/internal/compiler"
)

type useMap struct {
	m map[*Value]map[*Value]struct{}
}

// Add u to the users of v
// TODO: return a strongly typed something?
func (um useMap) Uses(v *Value) map[*Value]struct{} {
	if um.m[v] == nil {
		um.m[v] = make(map[*Value]struct{})
	}
	return um.m[v]
}

// Whether u uses v
/*
func (um uses) Uses(v, u *Value) bool{

}
*/

func gatherUses(um useMap, v *Value) {
	for _, a := range v.Args {
		um.Uses(a)[v] = struct{}{}
	}
	for _, a := range v.Args {
		gatherUses(um, a)
	}
}

type schedCtxt struct {
	sea              *compiler.Sea
	useMap           useMap
	regMap           map[*Value]regRange
	regNext          int
	reversedSchedule []*Value // TODO: should this be like a linked list?
	scheduled        map[*Value]struct{}
}

func (sched *schedCtxt) allUsesScheduled(v *Value) bool {
	for use := range sched.useMap.Uses(v) {
		if _, ok := sched.scheduled[use]; !ok {
			return false
		}
	}
	return true
}

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

// TODO: decouple scheduling and regalloc, there's actually not really a reason

func (sched *schedCtxt) value(v *Value) {
	if _, ok := sched.scheduled[v]; ok {
		return
	}

	if !sched.allUsesScheduled(v) {
		return
	}

	// TODO: should all pseudo instrs not be scheduled?
	sched.reversedSchedule = append(sched.reversedSchedule, v)
	sched.scheduled[v] = struct{}{}

	if _, ok := sched.regMap[v]; !ok {
		n := regs(v.Type)
		sched.regMap[v] = regRange{sched.regNext, n}
		sched.regNext += n // TODO: use a bitmap so we can allocate and free things
	}

	// Operands to a tuple must be assigned into consecutive registers.
	if v.Op == OpInterpreterPseudoMakeTuple {
		i := sched.regMap[v].i
		for argIndex, a := range v.Args {
			if _, alreadyAssigned := sched.regMap[a]; alreadyAssigned {
				// TODO: introduce renameMap instead of fucking around with
				// Value
				tmp := &compiler.Value{0, OpInterpreterCopy32, a.Type, []*Value{a}, nil}
				sched.useMap.Uses(a)[tmp] = struct{}{}
				sched.useMap.Uses(tmp)[v] = struct{}{}
				v.Args[argIndex] = tmp
				a = tmp
			}
			n := regs(a.Type)
			sched.regMap[a] = regRange{i, n}
			i += n
		}
	}

	for _, a := range v.Args {
		sched.value(a)
	}
}

func schedule(v *compiler.Value) ([]*Value, map[*Value]regRange) {
	um := useMap{make(map[*Value]map[*Value]struct{})}
	gatherUses(um, v)

	ctxt := schedCtxt{
		useMap:    um,
		regMap:    make(map[*Value]regRange),
		scheduled: make(map[*Value]struct{}),
	}
	ctxt.value(v)

	slices.Reverse(ctxt.reversedSchedule)

	return ctxt.reversedSchedule, ctxt.regMap
}
