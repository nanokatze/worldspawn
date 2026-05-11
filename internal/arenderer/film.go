package arenderer

type Film struct {
	Samples  []float32 // TODO: this should be something of a ring buffer
	Channels int
}
