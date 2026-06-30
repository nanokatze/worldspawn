package game

import "worldspawn/internal/gmath"

type AmmoType int8

// TODO: give this a better name pls
type WeaponHint struct {
	// TODO: this should choose the animation set basically. Or maybe we should
	// split this into two
	// Class string

	FirstPersonPropTransform gmath.TRS3f64
}

type WeaponButtons uint64

const (
	_ WeaponButtons = 1 << iota
	WeaponTrigger
)

// TODO: rename this to literally anything but
type Recoil struct {
	Recoil [2]float32
}
