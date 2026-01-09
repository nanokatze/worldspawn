package compositor

import "worldspawn/gpu"

const (
	OpStop = iota
)

type Pipeline struct {
	Program []uint32
}

// TODO: in the general case, could we decouple compositor from jq?
func (pipeline Pipeline) Run(jq *gpu.JobQueue) {
	// var state [256]any
	// for {
	// 	pc := 0
	// 	switch pipeline.Program[pc] {

	// 	}
	// }
}
