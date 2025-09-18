package material

import "worldspawn/internal/renderer/internal/material"

type Op = material.Op

var (
	OpFAdd = material.OpFAdd
	OpFSub = material.OpFSub
	OpFMul = material.OpFMul
	OpFDiv = material.OpFDiv
	OpFMin = material.OpFMin
	OpFMax = material.OpFMax

	OpFFloor = material.OpFFloor

	// OpSampleTexture = material.OpSampleTexture

	OpBSDFDiffuse = material.OpBSDFDiffuse

	OpXDFAdd   = material.OpXDFAdd
	OpXDFScale = material.OpXDFScale
)

type Value = material.Value
