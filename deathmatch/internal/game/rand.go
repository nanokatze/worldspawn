package game

import (
	"math/rand/v2"

	"github.com/go-json-experiment/json"
	"golang.org/x/crypto/blake2b"
)

// TODO: speed this thing up
func Rand(seed ...any) *rand.Rand {
	hasher, _ := blake2b.New(32, nil)
	if err := json.MarshalWrite(hasher, &seed); err != nil {
		panic(err)
	}
	return rand.New(rand.NewChaCha8([32]byte(hasher.Sum(nil))))
}
