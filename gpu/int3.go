package gpu

import "worldspawn/gpu/vk"

func int3FromVkOffset3D(from vk.Offset3D) [3]int {
	return [3]int{int(from.X), int(from.Y), int(from.Z)}
}

func int3FromVkExtent3D(from vk.Extent3D) [3]int {
	return [3]int{int(from.Width), int(from.Height), int(from.Depth)}
}

func mul3(x, y [3]int) [3]int {
	return [3]int{
		x[0] * y[0],
		x[1] * y[1],
		x[2] * y[2],
	}
}

func sub3(x, y [3]int) [3]int {
	return [3]int{
		x[0] - y[0],
		x[1] - y[1],
		x[2] - y[2],
	}
}

func mod3(x, y [3]int) [3]int {
	return [3]int{
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

func minify3(x [3]int, p int) [3]int {
	return [3]int{
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
