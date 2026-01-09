package core

import "worldspawn/internal/compiler"

var OpCondSelect = defOp("CondSelect", nil)

func CondSelect(b *compiler.Builder, x, y, cond *compiler.Class) *compiler.Class {
	return b.Value2(OpCondSelect, x.Type(), nil, x, y, cond)
}
