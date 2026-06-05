package game

import "worldspawn/internal/gmath"

var forward = gmath.Vec3f32{0, 1, 0}

// TODO: name these after planes, i.e. e01 should probably be named horizon and
// e12 meridian or idk.

var e01 = gmath.Rot3AToB(gmath.Vec3f32{1, 0, 0}, gmath.Vec3f32{0, 1, 0})
var e12 = gmath.Rot3AToB(gmath.Vec3f32{0, 1, 0}, gmath.Vec3f32{0, 0, 1})
