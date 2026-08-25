package bitslice

func divRoundUp(x, y int) int {
	return (x + y - 1) / y
}
