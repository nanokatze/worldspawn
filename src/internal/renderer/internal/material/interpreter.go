package material

// TODO: make an interpreter generator

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

// TODO: rename these enums to make it clear that it's some kind of "id" used by
// interpreter
type BSDF int8

const (
	_ BSDF = iota
	BSDFDiffuse
	BSDFMicrofacetGGXTest
)

type EDF int8

const (
	_ EDF = iota
	EDFUniform
)

// TODO: this ideally would be hashable for mc. We can either make this hashable
// somehow, or define functions to de/serialize this into a string.
type InterpretedMaterialOutputLayout struct {
	// TODO: BSDF and EDF are per-surface (and there can be two: front face and
	// back face). Beside that we also need VDFs and also AOVs.
	BSDFOff int
	BSDFs   []BSDF
	EDFOff  int
	EDFs    []EDF
}
