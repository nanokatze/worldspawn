package renderer

import (
	"bytes"
	"encoding/binary"

	"worldspawn/gpu"
	"worldspawn/internal/renderer/internal/material"
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

// TODO: unify this to consume either interpreted or SPIR-V material? We'll have
// to include the magic into the blob in that case.
func NewInterpretedMaterial(blob []byte) *InterpretedMaterial {
	r := bytes.NewReader(blob)

	var abi material.InterpreterABI
	binary.Read(r, binary.LittleEndian, &abi)

	code := gpu.MakeSliceUncached[uint32](r.Len() / 4)
	binary.Read(r, binary.LittleEndian, code.Value())

	return &InterpretedMaterial{
		emissive: abi.EDFCount > 0,
		program: material.InterpreterProgram{
			ABI:  abi,
			Code: gpu.SliceData(code),
		},
	}
}
