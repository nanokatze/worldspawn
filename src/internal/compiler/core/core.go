package core

import "worldspawn/internal/compiler"

func defOp(name string, validation compiler.ValidationFunc) compiler.Op {
	return compiler.DefOp("core."+name, validation)
}
