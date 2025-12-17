package gpu

import (
	"math/bits"
	"testing"

	"worldspawn/gpu/vk"
)

func TestImageExtent(t *testing.T) {
	img := NewImage(&ImageConfig{
		Dim:       ImageDim2D,
		Extent:    [3]int{15, 15, 1},
		Layers:    1,
		MipLevels: completeMipChainLength(15, 15, 1), // TODO: have an option for this?
		Samples:   1,
		Format:    vk.FORMAT_BC7_SRGB_BLOCK,
	})
	defer img.Destroy()

	for i := range 4 {
		t.Log(img.SubImage(img.Dim(), img.Format(), 0, 1, i, i+1).Extent())
		t.Log(img.SubImage(img.Dim(), vk.FORMAT_R32G32B32A32_UINT, 0, 1, i, i+1).Extent())
	}
}

func TestImageCopy1(t *testing.T) {
	a := NewImage(&ImageConfig{
		Dim:       ImageDim2D,
		Extent:    [3]int{4, 4, 1},
		Layers:    1,
		MipLevels: 1,
		Samples:   1,
		Format:    vk.FORMAT_BC7_UNORM_BLOCK,
	})
	defer a.Destroy()

	b := NewImage(&ImageConfig{
		Dim:       ImageDim2D,
		Extent:    [3]int{1, 1, 1},
		Layers:    1,
		MipLevels: 1,
		Samples:   1,
		Format:    vk.FORMAT_R32G32B32A32_UINT,
	})
	defer b.Destroy()

	var jq JobQueue
	a.EnqueueInit(&jq)
	b.EnqueueInit(&jq)
	EnqueueCopyImage(&jq,
		a, [3]int{0, 0, 0},
		b, [3]int{0, 0, 0},
		[3]int{1, 1, 1})
	jq.WaitForIdle()
}

func TestImageCopy3(t *testing.T) {
	a := NewImage(&ImageConfig{
		Dim:       ImageDim2D,
		Extent:    [3]int{3, 3, 1},
		Layers:    1,
		MipLevels: 1,
		Samples:   1,
		Format:    vk.FORMAT_BC7_UNORM_BLOCK,
	})
	defer a.Destroy()

	b := NewImage(&ImageConfig{
		Dim:       ImageDim2D,
		Extent:    [3]int{4, 4, 1},
		Layers:    1,
		MipLevels: 1,
		Samples:   1,
		Format:    vk.FORMAT_BC7_UNORM_BLOCK,
	})
	defer b.Destroy()

	var jq JobQueue
	a.EnqueueInit(&jq)
	b.EnqueueInit(&jq)
	EnqueueCopyImage(&jq,
		a, [3]int{},
		b, [3]int{},
		[3]int{4, 4, 1})
	jq.WaitForIdle()
}

func TestImageCopy2(t *testing.T) {
	img := NewImage(&ImageConfig{
		Dim:       ImageDim2D,
		Extent:    [3]int{6, 6, 1},
		Layers:    1,
		MipLevels: 1,
		Samples:   1,
		Format:    vk.FORMAT_BC7_UNORM_BLOCK,
	})
	defer img.Destroy()

	img2 := img.SubImage(img.Dim(), vk.FORMAT_R32G32B32A32_UINT, 0, 1, 0, 1)

	tmp := MakeSliceUncached[byte](2 * 2 * 16)
	defer Free(UnsafePointer(SliceData(tmp)))

	clear(tmp.Value())

	var jq JobQueue
	img2.EnqueueInit(&jq)
	EnqueueCopyMemoryToImage(&jq,
		img, [3]int{0, 0, 0},
		tmp, 0, 0,
		[3]int{6, 6, 1})
	EnqueueCopyMemoryToImage(&jq,
		img2, [3]int{0, 0, 0},
		tmp, 0, 0,
		[3]int{2, 2, 1})
	jq.WaitForIdle()
}

func completeMipChainLength(width, height, depth uint32) int {
	return log2_32(max(width, height, depth)) + 1
}

func log2_32(x uint32) int {
	return 32 - 1 - bits.LeadingZeros32(x)
}
