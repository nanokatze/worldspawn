package compiler

var OpCondSelect = DefOp("CondSelect", nil)

func CondSelect(b *Builder, x, y, cond *Class) *Class {
	return b.Value2(OpCondSelect, x.Type(), nil, x, y, cond)
}
