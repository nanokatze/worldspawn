package pathtracer

import "worldspawn/gpu"

type Film struct {
	Extent [2]int

	Color *gpu.Image

	// Albedo *gpu.Image
	// Depth  *gpu.Image
	// Normal *gpu.Image
	// Motion *gpu.Image
	// AOVs   []*gpu.Image
}
