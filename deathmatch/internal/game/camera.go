package game

// TODO: we'll want to implement viewshake here somehow. Generally this will
// have to be a pile of values that video renderer will have to eat up I guess.
// If we manage to cook up something successful with postprocess we could
// specify postprocess program in the game code.

// TODO: introduce Camera component which will specify fov etc. I guess we could
// also just use Entity.
type Camera struct {
	FieldOfView float32
}
