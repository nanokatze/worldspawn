package core

import (
	"strconv"

	"worldspawn/internal/compiler"
)

type BitsType struct{ N int64 }

func (t BitsType) String() string { return "Bits[" + strconv.FormatInt(t.N, 10) + "]" }

var (
	Bits1   compiler.Type = BitsType{1}
	Bits8   compiler.Type = BitsType{8}
	Bits16  compiler.Type = BitsType{16}
	Bits32  compiler.Type = BitsType{32}
	Bits64  compiler.Type = BitsType{64}
	Bits96  compiler.Type = BitsType{96}
	Bits128 compiler.Type = BitsType{128}
)

var OpConst = defOp("Const",
	func(typ compiler.Type, imm any, args ...*compiler.Class) {
		_ = imm.(uint64)
	})

func Const(b *compiler.Builder, typ compiler.Type, imm uint64) *compiler.Class {
	return b.Value2(OpConst, typ, imm)
}

/*
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
*/

var OpPack = defOp("Pack",
	func(typ compiler.Type, imm any, args ...*compiler.Class) {
		if imm != nil {
			panic("imm must be nil")
		}

		bits := int64(0)
		for _, a := range args {
			bits += a.Type().(BitsType).N
		}
		if typ != (BitsType{bits}) {
			panic("bad")
		}
	})

func Pack(b *compiler.Builder, args ...*compiler.Class) *compiler.Class {
	n := int64(0)
	for _, a := range args {
		n += a.Type().(BitsType).N
	}
	return b.Value2(OpPack, BitsType{n}, nil, args...)
}

// typ(a0 >> imm)
//
// TODO: make Extract more general
var OpExtract = defOp("Extract",
	func(typ compiler.Type, imm any, args ...*compiler.Class) {
		if len(args) != 1 {
			panic("wrong args")
		}

		off := imm.(int64)

		if !(0 <= off && off+typ.(BitsType).N <= args[0].Type().(BitsType).N) {
			panic("bad")
		}

		// TODO: this is unnecessarily strict
		if off%32 != 0 || typ.(BitsType).N != 32 {
			panic("bad")
		}
	})

func Extract(b *compiler.Builder, et compiler.Type, v *compiler.Class, off int64) *compiler.Class {
	return b.Value2(OpExtract, et, off, v)
}

func init() {
	Rules = append(Rules,
		// TODO: make this rule more general
		compiler.RewriteRule{
			Comment: "forward definition",
			Pattern: &compiler.Pattern{Op: OpExtract, Args: []*compiler.Pattern{{Op: OpPack, ArgsDDD: true}}},
			Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
				extractOff := v.Imm().(int64)

				for pack := range v.Arg(0).Values() {
					if pack.Op() != OpPack {
						continue
					}

					off := int64(0)
					for _, a := range pack.Args() {
						if off == extractOff {
							if a.Type() == v.Type() {
								rr.Class(a)
								rr.Kill(v)
							}
							break
						}
						off += a.Type().(BitsType).N
					}
				}
			},
		},
		compiler.RewriteRule{
			Comment: "trivial extract",
			Pattern: &compiler.Pattern{Op: OpExtract, Args: []*compiler.Pattern{{}}},
			Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
				off := v.Imm().(int64)

				if off == 0 && v.Type() == v.Arg(0).Type() {
					rr.Class(v.Arg(0))
					rr.Kill(v)
				}
			},
		})
}
