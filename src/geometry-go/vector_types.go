package geometry

import (
	"math"
)

// TODO: generate code for these and each vector type into a different file
// TODO: make these be array typedefs instead of struct.
// TODO: split this package into 32-bit and 64-bit types? Ehh

// TODO: I really feel like the renderer needs its own math package... We
// perhaps should just pretend that we're using the renderer's math package. Or
// IDK. Maybe indeed that's how we should do it, and the game then adds its own
// math as necessary?..
//
// Or perhaps this package should stay as a game's math lib and renderer should
// just use its own math lib, which will include things for stats, etc.

// TODO: game and renderer should probably use their own and separate math packages.

// TODO: make these structs again

type Vec2 struct{ X, Y float32 }

func (a Vec2) Add(b Vec2) Vec2 {
	return Vec2{X: a.X + b.X, Y: a.Y + b.Y}
}

func (a Vec2) Mul(b Vec2) Vec2 {
	return Vec2{X: a.X * b.X, Y: a.Y * b.Y}
}

func (a Vec2) Scale(lambda float32) Vec2 {
	a.X *= lambda
	a.Y *= lambda
	return a
}

func (a Vec2) Dot(b Vec2) float32 {
	return a.X*b.X + a.Y*b.Y
}

func (a Vec2) LengthSq() float32 {
	return a.Dot(a)
}

// TODO: rename to Norm and LengthSq to Norm2?
func (a Vec2) Length() float32 {
	return float32(math.Sqrt(float64(a.Dot(a))))
}

type Vec3 [3]float32

// TODO: rename these constructors to something else
func Vec3Broadcast(v float32) Vec3 {
	return Vec3{v, v, v}
}

func (a Vec3) Add(b Vec3) Vec3 {
	return Vec3{a[0] + b[0], a[1] + b[1], a[2] + b[2]}
}

func (a Vec3) Sub(b Vec3) Vec3 {
	return Vec3{a[0] - b[0], a[1] - b[1], a[2] - b[2]}
}

func (a Vec3) Scale(lambda float32) Vec3 {
	for i := range a {
		a[i] *= lambda
	}
	return a
}

func (a Vec3) Dot(b Vec3) (ab float32) {
	for i := range a {
		ab += a[i] * b[i]
	}
	return ab
}

func (a Vec3) Cross(b Vec3) (ab Vec3) {
	for i := 0; i < 3; i++ {
		ab[i] = a[(i+1)%3]*b[(i+2)%3] + a[(i+2)%3]*b[(i+1)%3]
	}
	return ab
}

func (a Vec3) LengthSq() float32 {
	return a.Dot(a)
}

func (a Vec3) Lerp(b Vec3, t float32) Vec3 {
	return Vec3{lerp32(a[0], b[0], t), lerp32(a[1], b[1], t), lerp32(a[2], b[2], t)}
}

func (a Vec3) Length() float32 {
	return float32(math.Sqrt(float64(a.LengthSq())))
}

// TODO: rename to Normalize? NormalizeOr? Maybe remove the Or argument as we
// basically never need it, and return either zero or a as-is (cycles). Remix
// actually has NormalizeOr but calls it safeNormalize.
func (a Vec3) NormalizedOr(b Vec3) Vec3 {
	lenSq := a.LengthSq()
	if lenSq == 0 {
		return b
	}
	return a.Scale(1 / float32(math.Sqrt(float64(lenSq))))
}

type Vec4 [4]float32

// TODO: rename to Splat or smth, intead of broadcast?
func Vec4Broadcast(v float32) Vec4 {
	return Vec4{v, v, v, v}
}

func (a Vec4) Scale(lambda float32) Vec4 {
	for i := range a {
		a[i] *= lambda
	}
	return a
}

func (a Vec4) Dot(b Vec4) (ab float32) {
	for i := range a {
		ab += a[i] * b[i] // TODO: fma
	}
	return ab
}

func (a Vec4) LengthSq() float32 {
	return a.Dot(a)
}

func (a Vec4) NormalizedOr(b Vec4) Vec4 {
	lenSq := a.LengthSq()
	if lenSq == 0 {
		return b
	}
	return a.Scale(1 / float32(math.Sqrt(float64(lenSq))))
}

type DVec3 [3]float64 // struct { X, Y, Z, _ float64 }

// var _ encoding.TextMarshaler = DVec3{}
// var _ encoding.TextUnmarshaler = (*DVec3)(nil)

func DVec3FromVec3(v Vec3) DVec3 {
	return DVec3{float64(v[0]), float64(v[1]), float64(v[2])}
}

func (a DVec3) Add(b DVec3) DVec3 {
	return DVec3{a[0] + b[0], a[1] + b[1], a[2] + b[2]}
}

func (a DVec3) Sub(b DVec3) DVec3 {
	return DVec3{a[0] - b[0], a[1] - b[1], a[2] - b[2]}
}

func (a DVec3) Scale(lambda float64) DVec3 {
	for i := range a {
		a[i] *= lambda
	}
	return a
}

func (a DVec3) Dot(b DVec3) float64 {
	return a[0]*b[0] + a[1]*b[1] + a[2]*b[2]
}

func (a DVec3) Vec3() Vec3 {
	return Vec3{float32(a[0]), float32(a[1]), float32(a[2])}
}
