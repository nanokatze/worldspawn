package gpu

import (
	"math/bits"
	"testing"

	"worldspawn/gpu/vk"
)

func TestImageExtent(t *testing.T) {
	// TODO: introduce WithCompleteMipChain()?

	img := NewImage(vk.FORMAT_BC7_SRGB_BLOCK, []int{15, 15}, WithMips2(completeMipChainLength(15, 15, 1)))
	defer img.Destroy()

	for i := range 4 {
		t.Log(
			img.SubImage(WithMips{i, i + 1}).Extent(),
			img.SubImage(WithMips{i, i + 1}, WithFormat(vk.FORMAT_R32G32B32A32_UINT)).Extent())
	}
}

func TestImageCopy1(t *testing.T) {
	a := NewImage(vk.FORMAT_BC7_UNORM_BLOCK, []int{4, 4})
	defer a.Destroy()

	b := NewImage(vk.FORMAT_R32G32B32A32_UINT, []int{1, 1})
	defer b.Destroy()

	var jq JobQueue
	a.EnqueueInit(&jq)
	b.EnqueueInit(&jq)
	EnqueueCopyImage(&jq,
		a, nil,
		b, nil,
		[]int{1, 1})
	WaitForIdle(&jq)
}

func TestImageCopy3(t *testing.T) {
	a := NewImage(vk.FORMAT_BC7_UNORM_BLOCK, []int{3, 3})
	defer a.Destroy()

	b := NewImage(vk.FORMAT_BC7_UNORM_BLOCK, []int{4, 4})
	defer b.Destroy()

	var jq JobQueue
	a.EnqueueInit(&jq)
	b.EnqueueInit(&jq)
	EnqueueCopyImage(&jq,
		a, nil,
		b, nil,
		[]int{4, 4})
	WaitForIdle(&jq)
}

func TestImageCopy2(t *testing.T) {
	img := NewImage(vk.FORMAT_BC7_UNORM_BLOCK, []int{6, 6})
	defer img.Destroy()

	img2 := img.SubImage(WithFormat(vk.FORMAT_R32G32B32A32_UINT))

	tmp := MakeSliceUncached[byte](2 * 2 * 16)
	defer Free(UnsafePointer(SliceData(tmp)))

	clear(tmp.Value())

	var jq JobQueue
	img2.EnqueueInit(&jq)
	EnqueueCopyMemoryToImage(&jq,
		img, nil,
		tmp, 0, 0,
		[]int{6, 6})
	EnqueueCopyMemoryToImage(&jq,
		img2, nil,
		tmp, 0, 0,
		[]int{2, 2})
	WaitForIdle(&jq)
}

func completeMipChainLength(width, height, depth uint32) int {
	return log2_32(max(width, height, depth)) + 1
}

func log2_32(x uint32) int {
	return 32 - 1 - bits.LeadingZeros32(x)
}
