package core

import (
	"fmt"

	"worldspawn/internal/compiler"
)

// TODO: rename Array to Vector?

type ArrayType struct {
	N int64
	T compiler.Type
}

func (t ArrayType) String() string { return fmt.Sprintf("Array[%d, %v]", t.N, t.T) }

var OpMakeArray = defOp("MakeArray", validateMakeArray)

// TODO: make this take elem type explicitly?
func MakeArray(b *compiler.Builder, et compiler.Type, args ...*compiler.Class) *compiler.Class {
	t := ArrayType{int64(len(args)), et}
	return b.Value2(OpMakeArray, t, nil, args...)
}

var OpArrayExtract = defOp("ArrayExtract", validateArrayExtract)

func ArrayExtract(b *compiler.Builder, arr *compiler.Class, idx int64) *compiler.Class {
	return b.Value2(OpArrayExtract, arr.Type().(ArrayType).T, idx, arr)
}

func init() {
	Rules = append(Rules,
		compiler.RewriteRule{
			Name:    "Forward ArrayExtract to the element's definition",
			Pattern: &compiler.Pattern{Op: OpArrayExtract, Args: []*compiler.Pattern{{Op: OpMakeArray, ArgsDDD: true}}},
			Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
				idx := v.Imm().(int64)
				for arr := range v.Arg(0).Values() {
					if arr.Op() == OpMakeArray {
						rr.Class(arr.Arg(int(idx)))
					}
				}
				// TODO: explain why
				rr.Kill(v)
			},
		},
		compiler.RewriteRule{
			Name:    "Split CondSelect of arrays into per element CondSelect",
			Pattern: &compiler.Pattern{Op: OpCondSelect, Args: []*compiler.Pattern{{}, {}, {}}},
			Rewrite: func(rr *compiler.RewriteResult, b *compiler.Builder, v *compiler.Value) {
				arr, ok := v.Type().(ArrayType)
				if !ok {
					return
				}

				x := v.Arg(0)
				y := v.Arg(1)
				cond := v.Arg(2)

				elems := make([]*compiler.Class, arr.N)
				for i := range arr.N {
					x_i := ArrayExtract(b, x, i)
					y_i := ArrayExtract(b, y, i)
					elems[i] = CondSelect(b, x_i, y_i, cond)
				}
				rr.Add2(OpMakeArray, arr, nil, elems...)
				// TODO: explain why
				rr.Kill(v)
			},
		})
}

func validateMakeArray(typ compiler.Type, imm any, args ...*compiler.Class) {
	if imm != nil {
		panic("imm must be nil")
	}

	arr := typ.(ArrayType)
	if len(args) != int(arr.N) {
		panic("mismatched len")
	}
	elem := arr.T
	for _, a := range args {
		if a.Type() != elem {
			panic("mismatched types")
		}
	}
}

func validateArrayExtract(typ compiler.Type, imm any, args ...*compiler.Class) {
	if len(args) != 1 {
		panic("wrong args")
	}

	idx := imm.(int64)

	arr := args[0]

	arrType := arr.Type().(ArrayType)

	if arrType.T != typ {
		panic("type mismatch")
	}
	if !(0 <= idx && idx < arrType.N) {
		panic("index out of bounds")
	}
}
