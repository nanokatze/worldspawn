package compiler

import "strconv"

type IntType struct{ N int64 }

func (t IntType) String() string { return "Int[" + strconv.FormatInt(t.N, 10) + "]" }

var (
	Int1  Type = IntType{1}
	Int8  Type = IntType{8}
	Int16 Type = IntType{16}
	Int32 Type = IntType{32}
)

// TODO: prefix all ops here with I?

var OpConst = DefOp("Const",
	func(typ Type, imm any, args ...*Class) {
		_ = imm.(int64)
	})

func Const(b *Builder, typ Type, imm int64) *Class {
	return b.Value2(OpConst, typ, imm)
}

var (
	OpAnd = DefOp("And", nil)
	OpOr  = DefOp("Or", nil)
	OpXor = DefOp("Xor", nil)
)

var OpNot = DefOp("Not", nil)

var (
	OpEqual    = DefOp("Equal", nil)
	OpNotEqual = DefOp("NotEqual", nil)
)

// TODO: move these into int.go
// OpIAdd = DefOp("IAdd")
// OpISub = DefOp("ISub")
// OpIMul = DefOp("IMul")
