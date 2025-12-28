package material

import (
	"structs"

	"worldspawn/gpu"
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

// TODO: rename from InterpreterOutputInterface
// TODO: once we plop this behind a pointer, change this to be a variable length
// byte bag
type InterpreterABI struct {
	BSDFs     [4]BSDF
	BSDFCount uint8
	BSDFsOff  uint8

	EDFs     [1]EDF
	EDFCount uint8
	EDFsOff  uint8

	// TODO: AOVs

	OutputsReg uint32
}

type InterpreterProgram struct {
	_ structs.HostLayout

	// TODO: plop this behind a pointer
	ABI InterpreterABI

	// TODO: could be a slice
	// TODO: rename to describe what this actually computes?
	Code gpu.Pointer[uint32]
}

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

	ALoadAttribute
	ALoadShadingNormal

	// ABSDFAlbedo
)
