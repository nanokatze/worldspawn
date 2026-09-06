package gpu

import "worldspawn/gpu/vk"

type point [maxDimensions]int

func pointZero() point {
	return point{}
}

func pointOne() point {
	return point{1, 1, 1}
}

func offset3(x []int) point {
	tmp := pointZero()
	copy(tmp[:], x)
	return tmp
}

func extent3(x []int) point {
	tmp := pointOne()
	copy(tmp[:], x)
	return tmp
}

func int3FromVkOffset3D(x vk.Offset3D) point {
	return point{int(x.X), int(x.Y), int(x.Z)}
}

func int3FromVkExtent3D(x vk.Extent3D) point {
	return point{int(x.Width), int(x.Height), int(x.Depth)}
}

func (x point) Sub(y point) point {
	return point{
		x[0] - y[0],
		x[1] - y[1],
		x[2] - y[2],
	}
}

func (x point) Mul(y point) point {
	return point{
		x[0] * y[0],
		x[1] * y[1],
		x[2] * y[2],
	}
}

func (x point) Mod(y point) point {
	return point{
		x[0] % y[0],
		x[1] % y[1],
		x[2] % y[2],
	}
}

func min3(x, y [3]int) [3]int {
	return [3]int{
		min(x[0], y[0]),
		min(x[1], y[1]),
		min(x[2], y[2]),
	}
}

func minify3(x point, p int) point {
	return point{
		minify(x[0], p),
		minify(x[1], p),
		minify(x[2], p),
	}
}

func vkOffset3DFromInt3(from [3]int) vk.Offset3D {
	return vk.Offset3D{X: int32(from[0]), Y: int32(from[1]), Z: int32(from[2])}
}

func vkExtent3DFromInt3(from [3]int) vk.Extent3D {
	return vk.Extent3D{Width: uint32(from[0]), Height: uint32(from[1]), Depth: uint32(from[2])}
}
