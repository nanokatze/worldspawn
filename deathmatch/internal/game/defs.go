package game

import "worldspawn/internal/gmath"

var (
	forward = gmath.Vec3f32{0, 1, 0}
	right   = gmath.Vec3f32{1, 0, 0}
	up      = gmath.Vec3f32{0, 0, 1}
)

// TODO: name these after planes, i.e. e01 should probably be named
// horzion, or horizonCCW.

var e01 = gmath.Rot3AToB(gmath.Vec3f32{1, 0, 0}, gmath.Vec3f32{0, 1, 0})
var e12 = gmath.Rot3AToB(gmath.Vec3f32{0, 1, 0}, gmath.Vec3f32{0, 0, 1})
