package pathtracer

import "worldspawn/gpu"

type Film struct {
	Extent [2]int

	// TODO: demodulate this for denoiser
	Color *gpu.Image

	// Albedo *gpu.Image
	// Depth  *gpu.Image
	Normal *gpu.Image
	// Motion *gpu.Image

	// AOVs []*gpu.Image
}

type _Film struct {
	Color  gpu.ImageDescriptors
	Normal gpu.ImageDescriptors
}
