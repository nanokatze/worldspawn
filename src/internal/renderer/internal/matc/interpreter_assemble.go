package matc

import (
	"reflect"
	"structs"
	"worldspawn/gpu"
	"worldspawn/internal/renderer/internal/compiler"
	"worldspawn/internal/renderer/internal/material"
)

type assembler struct {
	params map[string]uint32
	code   []uint32
}

// TODO: move to a different file? perhaps back to mc?
func packinstr(op material.A, dst, src0, src1 uint32) uint32 {
	return uint32(op) | uint32(dst)<<8 | uint32(src0)<<16 | uint32(src1)<<24
}

// Horrible hack, TODO: kill
type MaterialParams struct {
	_ structs.HostLayout

	// TODO: move code + output layout behind a pointer here as well
	Code gpu.Pointer[uint32]

	BSDFs     [4]uint8
	BSDFCount uint8
	BSDFsOff  uint8

	EDFs     [1]uint8
	EDFCount uint8
	EDFsOff  uint8

	OutputsReg uint32

	Triangles    gpu.Pointer[[3]uint16]
	NumTriangles uint32
	PosBuffer    gpu.Pointer[[3]float32]
	Normals      gpu.Pointer[[3]float32]
	UVs          gpu.Pointer[[2]float32]
	BaseColorR   float32
	BaseColorG   float32
	BaseColorB   float32
	Emission     [3]float32
}

func assemble(schedule []*compiler.Class, regm map[*compiler.Class]regRange) []uint32 {
	as := assembler{}

	as.params = map[string]uint32{}
	{
		typ := reflect.TypeFor[MaterialParams]()
		for i := range typ.NumField() {
			f := typ.Field(i)
			as.params[f.Name] = uint32(f.Offset)
		}
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
