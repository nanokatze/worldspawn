package game

import (
	"encoding/json/v2"
	"math/rand/v2"

	"golang.org/x/crypto/blake2b"
)

// TODO: speed this thing up. Use faster serializer, faster hash (e.g. blake3 or
// something that's also good at short hashes) and rand.NewPCG.
func Rand(seed ...any) *rand.Rand {
	hasher, _ := blake2b.New(32, nil)
	if err := json.MarshalWrite(hasher, &seed); err != nil {
		panic(err)
	}
	return rand.New(rand.NewChaCha8([32]byte(hasher.Sum(nil))))
}
