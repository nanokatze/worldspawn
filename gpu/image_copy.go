package gpu

import (
	"runtime"

	"worldspawn/gpu/vk"
	"worldspawn/gpu/vk/formatutil"
)

// TODO: should rowLength and imageHeight arguments be in bytes rather than texels?
// TODO: rename rowLength and imageHeight into rowStride and image/layerStride?

// TODO: replace vk.{Offset,Extent}3D with [3]uint32?

// TODO: make image copy functions methods on the *Image?

func offset3(x []int) [3]int {
	tmp := [3]int{0, 0, 0}
	copy(tmp[:], x)
	return tmp
}

func extent3(x []int) [3]int {
	tmp := [3]int{1, 1, 1}
	copy(tmp[:], x)
	return tmp
}

type copyImageJob struct {
	dst                 *imageData
	dstSubresourceRange subresourceRange
	dstOffset           vk.Offset3D
	src                 *imageData
	srcSubresourceRange subresourceRange
	srcOffset           vk.Offset3D
	extent              vk.Extent3D
	queueFamilies       uint32
}

func EnqueueCopyImage(
	jq *JobQueue,
	dst *Image, dstOffset []int,
	src *Image, srcOffset []int,
	extent []int) {
	enqueueCopyImage(jq, dst, offset3(dstOffset), src, offset3(srcOffset), extent3(extent))
}

func enqueueCopyImage(
	jq *JobQueue,
	dst *Image, dstOffset [3]int,
	src *Image, srcOffset [3]int,
	extent [3]int) {
	extentBlocks := divByBlockExtentRoundUp(extent, src.format)
	dstAspects := formatutil.Aspects(dst.format)
	dstMip := dst.subresourceRange.FirstMip()
	dstOffsetBlocks := divByBlockExtent(dstOffset, dst.format)
	dstOffsetBase, _ := copyRectInTexels(
		dst.data.format, dst.data.extent,
		dstMip,
		dstOffsetBlocks, extentBlocks)
	dstFamilies := chooseQueueFamiliesForImageCopy(
		dst.data.format, dst.data.extent,
		dstAspects, dstMip,
		dstOffsetBlocks, extentBlocks)

	srcAspects := formatutil.Aspects(src.format)
	srcMip := src.subresourceRange.FirstMip()
	srcOffsetBlocks := divByBlockExtent(srcOffset, src.format)
	srcOffsetBase, extentBase := copyRectInTexels(
		src.data.format, src.data.extent,
		srcMip,
		srcOffsetBlocks, extentBlocks)
	srcFamilies := chooseQueueFamiliesForImageCopy(
		src.data.format, src.data.extent,
		srcAspects, srcMip,
		srcOffsetBlocks, extentBlocks)

	families := dstFamilies & srcFamilies

	jq.Enqueue(&copyImageJob{
		dst:                 dst.data,
		dstSubresourceRange: dst.subresourceRange,
		dstOffset:           vkOffset3DFromInt3(dstOffsetBase),
		src:                 src.data,
		srcSubresourceRange: src.subresourceRange,
		srcOffset:           vkOffset3DFromInt3(srcOffsetBase),
		extent:              vkExtent3DFromInt3(extentBase),
		queueFamilies:       families,
	})
}

func (job *copyImageJob) Info() JobInfo {
	return JobInfo{
		QueueFamilies: job.queueFamilies,
	}
}

func (job *copyImageJob) Exec(q *CommandQueue) {
	q.Commands(func(cb vk.CommandBuffer) {
		var pinner runtime.Pinner
		defer pinner.Unpin()

		region := &vk.ImageCopy2{
			SType:          vk.STRUCTURE_TYPE_IMAGE_COPY_2,
			SrcSubresource: job.srcSubresourceRange.VkImageSubresourceLayers(job.src.format),
			SrcOffset:      job.srcOffset,
			DstSubresource: job.dstSubresourceRange.VkImageSubresourceLayers(job.dst.format),
			DstOffset:      job.dstOffset,
			Extent:         job.extent,
		}
		pinner.Pin(region)

		vkFns.CmdCopyImage2(cb, &vk.CopyImageInfo2{
			SType:          vk.STRUCTURE_TYPE_COPY_IMAGE_INFO_2,
			SrcImage:       job.src.vkImage,
			SrcImageLayout: vk.IMAGE_LAYOUT_GENERAL,
			DstImage:       job.dst.vkImage,
			DstImageLayout: vk.IMAGE_LAYOUT_GENERAL,
			RegionCount:    1,
			PRegions:       region,
		})
	})
}

// TODO: equip Pointers in these copies with length?

type copyMemoryToImageJob struct {
	dst                 *imageData
	dstSubresourceRange subresourceRange
	dstOffset           vk.Offset3D
	src                 UnsafePointer
	srcRowLength        uint32
	srcImageHeight      uint32
	extent              vk.Extent3D
	queueFamilies       uint32
}

// Only the first mip level is copied.
//
// TODO: better comment
func EnqueueCopyMemoryToImage(
	jq *JobQueue,
	dst *Image, dstOffset []int,
	src Slice[byte], srcRowLength, srcImageHeight int,
	extent []int) {
	enqueueCopyMemoryToImage(jq, dst, offset3(dstOffset), src, srcRowLength, srcImageHeight, extent3(extent))
}

func enqueueCopyMemoryToImage(
	jq *JobQueue,
	dst *Image, dstOffset [3]int,
	src Slice[byte], srcRowLength, srcImageHeight int,
	extent [3]int) {
	// TODO: validation

	extentBlocks := divByBlockExtentRoundUp(extent, dst.format)

	dstAspects := formatutil.Aspects(dst.format)
	dstMip := dst.subresourceRange.FirstMip()
	dstOffsetBlocks := divByBlockExtent(dstOffset, dst.format)
	dstOffsetBase, extentBase := copyRectInTexels(
		dst.data.format, dst.data.extent,
		dstMip,
		dstOffsetBlocks, extentBlocks)

	families := chooseQueueFamiliesForImageCopy(
		dst.data.format, dst.data.extent,
		dstAspects, dstMip,
		dstOffsetBlocks, extentBlocks)

	// BUG: we need to exclude the transfer-only queue family if the src isn't
	// aligned to a 4-byte boundary.

	jq.Enqueue(&copyMemoryToImageJob{
		dst:                 dst.data,
		dstSubresourceRange: dst.subresourceRange,
		dstOffset:           vkOffset3DFromInt3(dstOffsetBase),
		src:                 UnsafePointer(SliceData(src)),
		srcRowLength:        uint32(srcRowLength),
		srcImageHeight:      uint32(srcImageHeight),
		extent:              vkExtent3DFromInt3(extentBase),
		queueFamilies:       families,
	})
}

func (job *copyMemoryToImageJob) Info() JobInfo {
	return JobInfo{
		QueueFamilies: job.queueFamilies,
	}
}

func (job *copyMemoryToImageJob) Exec(q *CommandQueue) {
	q.Commands(func(cb vk.CommandBuffer) {
		var pinner runtime.Pinner
		defer pinner.Unpin()

		srcBuffer, srcOffset := BufferAndOffset(job.src)

		region := &vk.BufferImageCopy2{
			SType:             vk.STRUCTURE_TYPE_BUFFER_IMAGE_COPY_2,
			BufferOffset:      srcOffset,
			BufferRowLength:   uint32(job.srcRowLength),
			BufferImageHeight: uint32(job.srcImageHeight),
			ImageSubresource:  job.dstSubresourceRange.VkImageSubresourceLayers(job.dst.format),
			ImageOffset:       job.dstOffset,
			ImageExtent:       job.extent,
		}
		pinner.Pin(region)

		vkFns.CmdCopyBufferToImage2(cb, &vk.CopyBufferToImageInfo2{
			SType:          vk.STRUCTURE_TYPE_COPY_BUFFER_TO_IMAGE_INFO_2,
			SrcBuffer:      srcBuffer,
			DstImage:       job.dst.vkImage,
			DstImageLayout: vk.IMAGE_LAYOUT_GENERAL,
			RegionCount:    1,
			PRegions:       region,
		})
	})
}

type copyImageToMemoryJob struct {
	dst                 UnsafePointer
	dstRowLength        uint32
	dstImageHeight      uint32
	src                 *imageData
	srcSubresourceRange subresourceRange
	srcOffset           vk.Offset3D
	extent              vk.Extent3D
	queueFamilies       uint32
}

func EnqueueCopyImageToMemory(
	jq *JobQueue,
	dst Slice[byte], dstRowLength, dstImageHeight int,
	src *Image, srcOffset []int,
	extent []int) {
	enqueueCopyImageToMemory(jq, dst, dstRowLength, dstImageHeight, src, offset3(srcOffset), extent3(extent))
}

func enqueueCopyImageToMemory(
	jq *JobQueue,
	dst Slice[byte], dstRowLength, dstImageHeight int,
	src *Image, srcOffset [3]int,
	extent [3]int) {
	extentBlocks := divByBlockExtentRoundUp(extent, src.format)

	srcAspects := formatutil.Aspects(src.format)
	srcMip := src.subresourceRange.FirstMip()
	srcOffsetBlocks := divByBlockExtent(srcOffset, src.format)
	srcOffsetBase, extentBase := copyRectInTexels(
		src.data.format, src.data.extent,
		srcMip,
		srcOffsetBlocks, extentBlocks)
	families := chooseQueueFamiliesForImageCopy(
		src.data.format, src.data.extent,
		srcAspects, srcMip,
		srcOffsetBlocks, extentBlocks)

	// BUG: we need to exclude the transfer-only queue family if the src isn't
	// aligned to a 4-byte boundary.

	jq.Enqueue(&copyImageToMemoryJob{
		dst:                 UnsafePointer(SliceData(dst)),
		dstRowLength:        uint32(dstRowLength),
		dstImageHeight:      uint32(dstImageHeight),
		src:                 src.data,
		srcSubresourceRange: src.subresourceRange,
		srcOffset:           vkOffset3DFromInt3(srcOffsetBase),
		extent:              vkExtent3DFromInt3(extentBase),
		queueFamilies:       families,
	})
}

func (job *copyImageToMemoryJob) Info() JobInfo {
	return JobInfo{
		QueueFamilies: job.queueFamilies,
	}
}

func (job *copyImageToMemoryJob) Exec(q *CommandQueue) {
	q.Commands(func(cb vk.CommandBuffer) {
		var pinner runtime.Pinner
		defer pinner.Unpin()

		dstBuffer, dstOffset := BufferAndOffset(job.dst)

		region := &vk.BufferImageCopy2{
			SType:             vk.STRUCTURE_TYPE_BUFFER_IMAGE_COPY_2,
			BufferOffset:      dstOffset,
			BufferRowLength:   uint32(job.dstRowLength),
			BufferImageHeight: uint32(job.dstImageHeight),
			ImageSubresource:  job.srcSubresourceRange.VkImageSubresourceLayers(job.src.format),
			ImageOffset:       job.srcOffset,
			ImageExtent:       job.extent,
		}
		pinner.Pin(region)

		vkFns.CmdCopyImageToBuffer2(cb, &vk.CopyImageToBufferInfo2{
			SType:          vk.STRUCTURE_TYPE_COPY_IMAGE_TO_BUFFER_INFO_2,
			SrcImage:       job.src.vkImage,
			SrcImageLayout: vk.IMAGE_LAYOUT_GENERAL,
			DstBuffer:      dstBuffer,
			RegionCount:    1,
			PRegions:       region,
		})
	})
}

// TODO: rename
// TODO: provide a mechanism for feedback to the user?
// TODO: pass granularities array, queueFamilies explicitly?
func chooseQueueFamiliesForImageCopy(
	format vk.Format, imageExtent [3]int,
	aspects vk.ImageAspectFlags, mip int,
	offsetBlocks, extentBlocks [3]int) uint32 {
	levelExtent := minify3(imageExtent, mip)
	levelExtentBlocks := divByBlockExtentRoundUp(levelExtent, format)

	families := queueFamilies.Mask(0b100)

	if aspects&vk.ImageAspectFlags(vk.IMAGE_ASPECT_DEPTH_BIT) != 0 {
		families &= queueFamilies.Mask(0b001)
	}
	if aspects&vk.ImageAspectFlags(vk.IMAGE_ASPECT_STENCIL_BIT) != 0 {
		families &= queueFamilies.Mask(0b001)
	}

	entire := offsetBlocks == [3]int{} && extentBlocks == levelExtentBlocks
	if !entire {
		for family := range ones32(families &^ queueFamilies.Mask(0b010)) {
			granularity := int3FromVkExtent3D(queueFamilies.props[family].MinImageTransferGranularity)

			if granularity == ([3]int{}) ||
				mod3(offsetBlocks, granularity) != ([3]int{}) ||
				mod3(extentBlocks, granularity) != ([3]int{}) {
				families &^= 1 << family
			}
		}
	}

	return families
}

func copyRectInTexels(
	format vk.Format, imageExtent [3]int,
	mip int,
	offsetBlocks, extentBlocks [3]int) ([3]int, [3]int) {
	blockExtent := formatBlockExtent(format)
	levelExtent := minify3(imageExtent, mip)

	offset := mul3(offsetBlocks, blockExtent)
	extent := min3(mul3(extentBlocks, blockExtent), sub3(levelExtent, offset))
	return offset, extent
}

// TODO: optimize for block sides? We don't need the general division here.

func divByBlockExtent(x [3]int, yFormat vk.Format) [3]int {
	y := formatBlockExtent(yFormat)
	return [3]int{
		x[0] / y[0],
		x[1] / y[1],
		x[2] / y[2],
	}
}

func divByBlockExtentRoundUp(x [3]int, yFormat vk.Format) [3]int {
	y := formatBlockExtent(yFormat)
	return [3]int{
		(x[0] + y[0] - 1) / y[0],
		(x[1] + y[1] - 1) / y[1],
		(x[2] + y[2] - 1) / y[2],
	}
}
