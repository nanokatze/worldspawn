package renderer

import "structs"

type Quality struct {
	_ structs.HostLayout

	MaxBounces int32

	RussianRouletteThreshold int32
}
