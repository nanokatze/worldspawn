package material

// TODO: it would be nice if we have base compiler and then extend it with more
// Ops, etc.

// TODO: figure out how to do types

type Op int

const (
	_ Op = iota

	OpAddF
	OpSubF
	OpMulF
	OpDivF
	OpMinF
	OpMaxF

	OpFracF // TODO: replace in favor of x - floor(x)
	OpFloorF

	OpSampleTexture

	OpBSDFDiffuse

	OpBSDFScale
	OpBSDFAdd
)

// TODO: rename to Instr?
type Value struct {
	Op Op
	// Type
	Args []*Value
	Aux  any
	// DI any
}

// type Func struct {
// 	Values []*Value
// }

// TODO: let Validate print things somewhere, e.g. slog.Logger?
// func Validate(v *Value)

func CompileInterpreterProgram(v *Value) []uint32 {
	return nil
}
