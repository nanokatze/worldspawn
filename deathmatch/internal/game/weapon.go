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
	DrawDuration time.Duration
	HideDuration time.Duration

	// TODO: this should choose the animation set basically. Or maybe we should
	// split this into more stuff
	// WeaponClass string

	FirstPersonPropTransform gmath.TRS3f64
}

type WeaponButtons uint64

// TODO: could we generalize this to arbitrary knobs? And have WeaponHint
// instruct the gladiator code how to operate them
const (
	_ WeaponButtons = 1 << iota
	WeaponTrigger
)
