package material

type A uint32

// TODO: make an interpreter generator instead of an extensible interpreter?

// TODO: prefix these differently
const (
	AStop A = iota

	ACopy32

	AConst32

	AFAdd32
	AFSub32
	AFMul32
	AFDiv32
	AFMin32
	AFMax32

	AFFloor32

	AFEqual32
	AFNotEqual32
	AFLessOrEqual32

	AConditionalSelect32

	ALoad          // TODO: rename
	ALoadAttribute // TODO: rename
	ALoadNormal
)

func packinstr(op A, dst, src0, src1 uint32) uint32 {
	return uint32(op) | uint32(dst)<<8 | uint32(src0)<<16 | uint32(src1)<<24
}
