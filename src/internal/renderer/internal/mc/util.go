package mc

import "worldspawn/internal/renderer/internal/compiler"

func buildArith1(b *compiler.Builder, op compiler.Op, x *compiler.Class) *compiler.Class {
	return b.Value2(op, x.Type(), nil, x)
}

func buildArith2(b *compiler.Builder, op compiler.Op, x, y *compiler.Class) *compiler.Class {
	return b.Value2(op, x.Type(), nil, x, y)
}
