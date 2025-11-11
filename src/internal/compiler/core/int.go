package core

// TODO: split int type and ops out of core? Or alternatively bring float into
// the core.

import (
	"strconv"

	"worldspawn/internal/compiler"
)

type IntType struct{ N int64 }

func (t IntType) String() string { return "Int[" + strconv.FormatInt(t.N, 10) + "]" }

var (
	Int1  compiler.Type = IntType{1}
	Int8  compiler.Type = IntType{8}
	Int16 compiler.Type = IntType{16}
	Int32 compiler.Type = IntType{32}
	Int64 compiler.Type = IntType{64}
)

// TODO: prefix all ops here with I?

var OpIConst = defOp("IConst",
	func(typ compiler.Type, imm any, args ...*compiler.Class) {
		_ = imm.(int64)
	})

func Const(b *compiler.Builder, typ compiler.Type, imm int64) *compiler.Class {
	return b.Value2(OpIConst, typ, imm)
}

var (
	OpAnd = defOp("And", nil)
	OpOr  = defOp("Or", nil)
	OpXor = defOp("Xor", nil)
)

var OpNot = defOp("Not", nil)

var (
	OpEqual    = defOp("Equal", nil)
	OpNotEqual = defOp("NotEqual", nil)
)

// OpIAdd = defOp("IAdd")
// OpISub = defOp("ISub")
// OpIMul = defOp("IMul")
