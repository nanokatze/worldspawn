package matc

import (
	"unsafe"

	"worldspawn/internal/compiler"
	"worldspawn/internal/renderer/internal/material"
)

type assembler struct {
	params []uint32 // can be just array at this point
	code   []uint32
}

// TODO: move to a different file? perhaps back to mc?
func packinstr(op material.A, dst, src0, src1 uint32) uint32 {
	return uint32(op) | uint32(dst)<<8 | uint32(src0)<<16 | uint32(src1)<<24
}

func assemble(schedule []*compiler.Class, regm map[*compiler.Class]regRange) []uint32 {
	as := assembler{}

	as.params = []uint32{
		uint32(unsafe.Offsetof(material.MaterialParams{}.UVs)),
		uint32(unsafe.Offsetof(material.MaterialParams{}.BaseColorR)),
		uint32(unsafe.Offsetof(material.MaterialParams{}.BaseColorG)),
		uint32(unsafe.Offsetof(material.MaterialParams{}.BaseColorB)),
	}

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
