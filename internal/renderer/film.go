package renderer

import "worldspawn/gpu"

type Film struct {
	Extent [2]int

	Color *gpu.Image

	DiffuseAlbedo      *gpu.Image
	NormalAndRoughness *gpu.Image
	Depth              *gpu.Image
	Motion             *gpu.Image
}

type _Film struct {
	Color gpu.ImageDescriptor

	DiffuseAlbedo      gpu.ImageDescriptor
	NormalAndRoughness gpu.ImageDescriptor
	Depth              gpu.ImageDescriptor
	Motion             gpu.ImageDescriptor
}
