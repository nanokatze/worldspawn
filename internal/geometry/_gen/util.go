package main

import (
	"math/big"
)

func binomial(n, k int64) int64 {
	var z big.Int
	z.Binomial(n, k)
	return z.Int64()
}
