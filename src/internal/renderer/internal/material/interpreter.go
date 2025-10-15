package material

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"worldspawn/internal/renderer/internal/compiler"
)

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

type A uint32

// TODO: make an interpreter generator instead of an extensible interpreter

// TODO: prefix these differently
const (
	AStop A = iota

	ACopy32
	AXchg32 // for parallel copies

	AConst32

	AFAddE8M23
	AFSubE8M23
	AFMulE8M23
	AFDivE8M23
	AFMinE8M23
	AFMaxE8M23

	AFFloorE8M23

	AFEqualE8M23
	AFLessOrEqualE8M23

	AConditionalSelect32

	ALoad          // TODO: rename
	ALoadAttribute // TODO: rename
	ALoadNormal

	// ABSDFAlbedo
)

func packinstr(op A, dst, src0, src1 uint32) uint32 {
	return uint32(op) | uint32(dst)<<8 | uint32(src0)<<16 | uint32(src1)<<24
}

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

// TODO: make private and apply this at CompileInterpreterProgram.
var LowerToInterpreter = []compiler.RewriteRule{
	{
		Pattern: &compiler.Pattern{
			Op:      OpMakeTuple,
			ArgsDDD: true,
		},
		Replace: func(sea *compiler.Sea, v *compiler.Value) *compiler.Value {
			return sea.Value(opInterpreterPseudoMakeTuple, v.Type(), nil, v.Args...)
		},
	},
	{
		Pattern: &compiler.Pattern{
			Op: OpTupleExtract,
			Args: []*compiler.Pattern{
				{},
			},
		},
		Replace: func(sea *compiler.Sea, v *compiler.Value) *compiler.Value {
			return sea.Value(opInterpreterPseudoTupleExtract, v.Type(), uint32(v.Imm().(int)), v.Args...)
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
				return sea.Value(opInterpreterConst32, v.Type(), uint32(imm))
			default:
				// TODO: panic with an error object instead
				panic(fmt.Sprintf("cannot lower %d-bit %s", bits, v.Op()))
			}
		},
	},
	shrimpleLowering(2, OpFSub, opInterpreterFSubE8M23),
	shrimpleLowering(2, OpFMin, opInterpreterFMinE8M23),
	shrimpleLowering(1, OpFFloor, opInterpreterFFloorE8M23),
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

func CompileInterpreterProgram(sea *compiler.Sea, c *compiler.Class) []uint32 {
	// TODO: lower v with LowerToInterpreter. We'd probably want to make a copy
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

	regm := regassign(sea, sched)

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
		case AConst32:
			data := assembled[i]
			i++
			fmt.Fprintf(os.Stderr, " 0x%08x", data)
		}
		fmt.Fprintln(os.Stderr)
	}
	return assembled
}
