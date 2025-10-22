package mc

// TODO: what should this file actually contain? I guess we need a file that
// implements the interpreter (well, we implement it in slang, but the go
// counterpart needs to define instructions). Then, we also need e.g. assembler
// and disassembler, teach the compiler how to compile to interpreter, etc.

// TODO: make an interpreter generator instead of an extensible interpreter

//go:generate stringer -type A -trimprefix A

type A uint32

// TODO: prefix these differently
const (
	AStop A = iota

	ACopy32

	AConst32

	AFAddE8M23
	AFSubE8M23
	AFMulE8M23
	AFDivE8M23

	AFMinE8M23
	AFMaxE8M23

	AFFloorE8M23

	AFEqualE8M23
	AFLessOrEqualE8M23

	ACondSelect32

	ALoadParam
	ALoadAttr
	ALoadNormal // GetGeometricNormal or whatever

	// ABSDFAlbedo
)

func packinstr(op A, dst, src0, src1 uint32) uint32 {
	return uint32(op) | uint32(dst)<<8 | uint32(src0)<<16 | uint32(src1)<<24
}
