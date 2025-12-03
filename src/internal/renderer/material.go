package renderer

import (
	"reflect"

	"worldspawn/gpu"
	"worldspawn/internal/renderer/internal/material"
	"worldspawn/internal/renderer/matc"
)

/*
type MaterialSet struct {
	pipeline *gpu.RayTracingPipeline
	sbt      gpu.ShaderBindingTable
}
*/

// NOTE: This is almost like matc.InterpretedMaterial. I guess we should just
// make InterpretedMaterial be an interface with two implementations. We'll have
// to cook up InterpretedMaterial which will be basically
// matc.InterpretedMaterial but with code being gpu.Slice[uint32].
//
// TODO: make this an implementation of Material interface, and return the interface, and make this private I guess.
type InterpretedMaterial struct {
	program material.InterpreterProgram

	// TODO: kill this. Currently we can't, the user should wrap renderer.Mesh,
	// but ...
	ParamStruct reflect.Type
	ParamNames  []string
}

func (m *InterpretedMaterial) emissive() bool {
	return m.program.ABI.EDFCount > 0
}

// TODO: make blob some other type so that we drop dependency on matc. Maybe
// []byte container or string or idk? In that case I guess we also could unify
// the entry point to consume either interpreted or compiled material, by
// distinguishing from the format specified in the container in the blob.
func NewInterpretedMaterial(blob *matc.InterpretedMaterial) *InterpretedMaterial {
	device := gpu.MakeSliceUncached[uint32](len(blob.Code))
	copy(device.Value(), blob.Code)

	return &InterpretedMaterial{
		program: material.InterpreterProgram{
			ABI:  blob.ABI,
			Code: gpu.SliceData(device),
		},
	}
}
