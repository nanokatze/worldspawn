package gpu

import (
	"worldspawn/gpu/vk"
	"worldspawn/gpu/vk/formatutil"
)

type Format = vk.Format

func formatBlockExtent(format Format) [3]int {
	return int3FromVkExtent3D(formatutil.Describe(format).BlockExtent)
}

type IndexType = vk.IndexType
