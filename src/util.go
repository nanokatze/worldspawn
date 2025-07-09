package worldspawn

import (
	"time"
)

func durationToFloatSeconds(d time.Duration) float64 {
	return float64(d) / 1e9
}
