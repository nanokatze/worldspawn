package pathtracer

import "worldspawn/gpu"

// TODO: should this be an interface so that we can support different texture
// types (e.g. some flavor of neural)?
type Texture struct {
	Image *gpu.Image
}
