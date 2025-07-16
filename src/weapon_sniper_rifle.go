package worldspawn

import (
	"math"
	"time"

	"worldspawn/ecs"
	"worldspawn/geometry-go"
)

// TODO: sniper rifle should have ambient noise
// TODO: draw the laser
// TODO: we should apply various charge-related effects (sound cue, the laser
// flash effect, player slowdown) past some charge %
// TODO: deal knockback to both the shooter (only when they're in air) and the
// target (all the time.) This will be useful as a mobility tool for the shooter
// as well as throw targets off and keep them away from the sniper
// TODO: spawn tracer effect after making a shot. Charged shot should spawn a
// more visually significant effect, so that players can easily distinguish
// between the two
// TODO: the sniper rifle should be able to penetrate thin surfaces and
// characters, with some impact reduction
// TODO: zoom
// TODO: distance impact multiplier params
// TODO: think about whether and how we want headshots (or other hitbox groups)
// to affect the damage and or falloff... I think we want headshots to be
// consistent and reward high tech gameplay, but they should be affected by
// damage falloff as usual. We may use an unusual multiplier for headshots on
// sniper.

type WeaponSniperRifle struct {
	BaseDamage        int32
	BaseSelfKnockback float32 // TODO: should be in Newtons
	// TODO: charge and headshot impact multipliers
	CycleDuration      time.Duration
	ChargeDuration     time.Duration
	OverchargeDuration time.Duration
	ShootSound         string
	ChargeSound        string
	ChargeReadySound   string

	ZoomRatios                     []float32
	ZoomWalkSpeedPenalty           float32
	ZoomCycleDuration              time.Duration
	ZoomCycleDurationAccessibility time.Duration
	ZoomSound                      string

	Charging        bool
	ChargeBeginTime Time

	// NotifiedChargeBegin bool
	NotifiedChargeReady bool

	NextAttack     Time
	NextZoomAdjust Time
}

func init() {
	registerEntity[WeaponSniperRifle]()
}

var _ WeaponUpdateInterface = WeaponSniperRifle{}

// TODO: change to value receiver? we might want the network differ to assume
// that the state only changed if it's different by value comparison.
func (weapon WeaponSniperRifle) WeaponUpdateSubtick(w *World, weaponID, operatorID ecs.ID, now Time, info *UpdateInfo) (recoil geometry.Vec3) {
	if w.Now < weapon.NextAttack {
		return
	}

	aim, _ := w.WeaponAim.Load(weaponID)

	switch {
	case !weapon.Charging && aim.Buttons&(1<<ButtonAttack) != 0:
		weapon.Charging = true
		weapon.ChargeBeginTime = w.Now
		weapon.NotifiedChargeReady = false

	case weapon.Charging && (aim.Buttons&(1<<ButtonAttack) == 0 || w.Now.Sub(weapon.ChargeBeginTime) >= weapon.OverchargeDuration):
		charge := float32(math.Min(math.Max(durationToFloatSeconds(w.Now.Sub(weapon.ChargeBeginTime))/durationToFloatSeconds(weapon.ChargeDuration), 0), 1))

		chargeImpactMultiplier := 1 + 2*charge

		w.SoundEffect.Store(weaponID, SoundEffect{
			Effect:   weapon.ShootSound,
			PlayTime: w.Now,
		})

		// TODO: adjust recoil by chargeImpactMultiplier
		// w.WeaponRecoil.Store(entityID, geometry.Vec3{X: 0.01})

		// Scaling knockback by sqrt of impact multiplier results feels nicer
		// than scaling by impact multiplier as is.
		selfKnockback := weapon.BaseSelfKnockback * chargeImpactMultiplier

		// Apply self-knockback.
		//
		// TODO: only apply knockback if we're not on ground
		//
		// TODO: plumb this through result so that we don't have to apply
		// knockback directly, but can delegate it to the player movement code
		playerVelocity, _ := w.Velocity.Load(operatorID)
		playerVelocity.Linear = playerVelocity.Linear.Add(aim.ShootRotation.Rotate(geometry.Vec3{0, selfKnockback, 0}))
		w.Velocity.Store(operatorID, playerVelocity)

		weapon.Charging = false
		weapon.NextAttack = w.Now.Add(weapon.CycleDuration)

	case weapon.Charging && w.Now.Sub(weapon.ChargeBeginTime) >= weapon.ChargeDuration && !weapon.NotifiedChargeReady:
		if weapon.ChargeReadySound != "" {
			w.SoundEffect.Store(weaponID, SoundEffect{
				Effect:   weapon.ChargeReadySound,
				PlayTime: w.Now,
			})
		}

		weapon.NotifiedChargeReady = true
	}

	w.Entity.Store(weaponID, weapon)
	return
}
