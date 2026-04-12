package matc

import (
	"worldspawn/internal/compiler"
	"worldspawn/internal/grenderer/internal/material"
)

type assembler struct {
	params ParamsTuple
	code   []uint32
}

// TODO: move to a different file? perhaps back to mc?
func packinstr(op material.A, dst, src0, src1 uint32) uint32 {
	return uint32(op) | uint32(dst)<<8 | uint32(src0)<<16 | uint32(src1)<<24
}

func assemble(schedule []*compiler.Class, regm map[*compiler.Class]regRange, paramStruct ParamsTuple) []uint32 {
	as := assembler{}
	as.params = paramStruct

	for _, class := range schedule {
		v := class.Value()

		f, ok := amap[v.Op()]
		if !ok {
			panic("can't assemble op " + v.Op().String())
		}
		f(&as, class, v, regm)
	}

	as.code = append(as.code, packinstr(material.AStop, 0, 0, 0))

	return as.code
}
