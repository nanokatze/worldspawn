package compiler

import "fmt"

// TODO: bring ArrayType into a usable state

type ArrayType struct {
	elem Type
	len  int64
}

// TODO: hash cons this stuff later
func MakeArrayType(elem Type, len int64) ArrayType {
	return ArrayType{elem, len}
}

func (typ ArrayType) Elem() Type { return typ.elem }

func (typ ArrayType) Len() int64 { return typ.len }

func (typ ArrayType) String() string { return fmt.Sprintf("[%d]%v", typ.len, typ.elem) }

var OpMakeArray = DefOp("MakeArray",
	func(typ Type, imm any, args ...*Class) {
		if imm != nil {
			panic("imm must be nil")
		}

		arr := typ.(ArrayType)
		if len(args) != int(arr.Len()) {
			panic("mismatched len")
		}
		elem := arr.Elem()
		for _, a := range args {
			if a.Type() != elem {
				panic("mismatched types")
			}
		}
	})

// TODO: make this take elem type explicitly?
func MakeArray(b *Builder, et Type, args ...*Class) *Class {
	t := MakeArrayType(et, int64(len(args)))
	return b.Value2(OpMakeArray, t, nil, args...)
}

var OpArrayExtract = DefOp("ArrayExtract",
	func(typ Type, imm any, args ...*Class) {
		if len(args) != 1 {
			panic("wrong args")
		}

		idx := imm.(int64)

		arr := args[0]

		arrType := arr.Type().(ArrayType)

		if arrType.Elem() != typ {
			panic("type mismatch")
		}
		if !(0 <= idx && idx < arrType.len) {
			panic("index out of bounds")
		}
	})

func ArrayExtract(b *Builder, arr *Class, idx int64) *Class {
	return b.Value2(OpArrayExtract, arr.Type().(ArrayType).Elem(), idx, arr)
}
