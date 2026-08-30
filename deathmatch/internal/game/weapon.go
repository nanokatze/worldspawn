package game

import (
	"time"

	"worldspawn/internal/gmath"
)

type AmmoType int8

const (
	_ AmmoType = iota - 1
	AmmoGrenades
	AmmoBullets
)

// TODO: give this a better name pls
type WeaponHint struct {
	DrawDurationMultiplier float32
	HideDurationMultiplier float32

	// TODO: this should choose the animation set basically. Or maybe we should
	// split this into more stuff
	// WeaponClass string

	FirstPersonPropTransform gmath.Affine3TRSf64
}

type WeaponButtons uint64

// TODO: could we generalize this to arbitrary knobs? And have WeaponHint
// instruct the gladiator code how to operate them
const (
	_ WeaponButtons = 1 << iota
	WeaponTrigger
)

const weaponBaseDrawDuration = 500 * time.Millisecond
const weaponBaseHideDuration = 500 * time.Millisecond
