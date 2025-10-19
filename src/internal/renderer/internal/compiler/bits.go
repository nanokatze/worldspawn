package compiler

import "strconv"

type BitsType struct{ N int64 }

func (t BitsType) String() string { return "Bits[" + strconv.FormatInt(t.N, 10) + "]" }

var (
	Bits1  Type = BitsType{1}
	Bits8  Type = BitsType{8}
	Bits16 Type = BitsType{16}
	Bits32 Type = BitsType{32}
)

var OpConst = DefOp("Const",
	func(typ Type, imm any, args ...*Class) {
		_ = imm.(int64)
	})

func BuildConst(b *Builder, typ Type, imm int64) *Class {
	return b.Value2(OpConst, typ, imm)
}

var OpCondSelect = DefOp("CondSelect", nil)

var OpEqual = DefOp("Equal", nil)

var (
	OpAnd = DefOp("And", nil)
	OpOr  = DefOp("Or", nil)
	OpXor = DefOp("Xor", nil)
)

var OpNot = DefOp("Not", nil)

// TODO: move this into a separate file?
// OpIAdd = DefOp("IAdd")
// OpISub = DefOp("ISub")
// OpIMul = DefOp("IMul")
