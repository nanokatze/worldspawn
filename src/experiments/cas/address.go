package cas

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type Address [32]byte

func AddressFromString(s string) (Address, error) {
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return Address{}, err
	}
	if len(bytes) != len(Address{}) {
		// TODO: better error pls
		return Address{}, fmt.Errorf("bad length")
	}
	return Address(bytes), nil
}

func AddressOf(content []byte) Address {
	return Address(sha256.Sum256(content))
}

func (addr Address) String() string { return fmt.Sprintf("%x", addr[:]) }
