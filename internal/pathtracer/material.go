package pathtracer

import (
	"worldspawn/gpu"
	"worldspawn/internal/pathtracer/internal/material"
	"worldspawn/internal/pathtracer/matc"
)

/*
type MaterialSet struct {
	pipeline *gpu.RayTracingPipeline
	sbt      gpu.ShaderBindingTable
}
*/

// TODO: rename Material interface to MaterialProgram? idk tbh it's a bit uhh
// uhhhh

// TODO: make this an implementation of Material interface, and return the
// interface, and make this private I guess.
// TODO: in addition to emissive flag, should we equip materials with a host
// function to refine whether the material should be added to the light accel
// based on what's in the args? This could also be made the user's
// responsibility by introducing some way to control mesh part's inclusion into
// the light accel.
type InterpretedMaterial struct {
	emissive bool // TODO: replace with summary flags? possibly move into material.InterpreterProgram?
	program  material.InterpreterProgram
}

// TODO: make blob some other type so that we drop dependency on matc. Maybe
// []byte container or string or idk? In that case I guess we also could unify
// the entry point to consume either interpreted or compiled material, by
// distinguishing from the format specified in the container in the blob.
func NewInterpretedMaterial(blob *matc.InterpretedMaterial) *InterpretedMaterial {
	device := gpu.MakeSliceUncached[uint32](len(blob.Code))
	copy(device.Value(), blob.Code)

	return &InterpretedMaterial{
		emissive: blob.ABI.EDFCount > 0,
		program: material.InterpreterProgram{
			ABI:  blob.ABI,
			Code: gpu.SliceData(device),
		},
	}
}
