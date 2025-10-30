package core

import "worldspawn/internal/renderer/internal/compiler"

var OpCondSelect = compiler.DefOp("CondSelect", nil)

func CondSelect(b *compiler.Builder, x, y, cond *compiler.Class) *compiler.Class {
	return b.Value(OpCondSelect, x.Type(), x, y, cond)
}
