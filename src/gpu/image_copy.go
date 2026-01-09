package gpu

import (
	"runtime"

	"worldspawn/gpu/vk"
	"worldspawn/gpu/vk/formatutil"
)

// TODO: should rowLength and imageHeight arguments be in bytes rather than texels?
// TODO: rename rowLength and imageHeight into rowStride and image/layerStride?

type copyImageJob struct {
	dst           *imageData
	dstAspects    vk.ImageAspectFlags
	dstBaseLayer  uint32
	dstLayers     uint32
	dstMipLevel   uint8
	dstOffset     vk.Offset3D
	src           *imageData
	srcAspects    vk.ImageAspectFlags
	srcBaseLayer  uint32
	srcLayers     uint32
	srcMipLevel   uint8
	srcOffset     vk.Offset3D
	extent        vk.Extent3D
	queueFamilies uint32
}

func EnqueueCopyImage(
	jq *JobQueue,
	dst *Image, dstOffset [3]int,
	src *Image, srcOffset [3]int,
	extent [3]int) {
	dstAspects := formatutil.Aspects(dst.format)
	dstMipLevel := int(dst.baseMipLevel)
	dstOffsetBlocks := int3(dstOffset).Div(formatBlockExtent(dst.format))
	srcAspects := formatutil.Aspects(src.format)
	srcMipLevel := int(src.baseMipLevel)
	srcOffsetBlocks := int3(srcOffset).Div(formatBlockExtent(src.format))
	extentBlocks := divRoundUp3(extent, formatBlockExtent(src.format))

	dstOffsetBase, _ := copyRectInTexels(
		dst.base.format, int3FromVkExtent3D(dst.base.extent),
		dstMipLevel,
		dstOffsetBlocks, extentBlocks)
	srcOffsetBase, extentBase := copyRectInTexels(
		src.base.format, int3FromVkExtent3D(src.base.extent),
		srcMipLevel,
		srcOffsetBlocks, extentBlocks)

	dstFamilies := chooseQueueFamiliesForImageCopy(
		dst.base.format, int3FromVkExtent3D(dst.base.extent),
		dstAspects, dstMipLevel,
		dstOffsetBlocks, extentBlocks)
	srcFamilies := chooseQueueFamiliesForImageCopy(
		src.base.format, int3FromVkExtent3D(src.base.extent),
		srcAspects, srcMipLevel,
		srcOffsetBlocks, extentBlocks)
	families := dstFamilies & srcFamilies

	jq.Enqueue(&copyImageJob{
		dst:           dst.base,
		dstAspects:    dstAspects,
		dstBaseLayer:  dst.baseLayer,
		dstLayers:     dst.layers,
		dstMipLevel:   uint8(dstMipLevel),
		dstOffset:     vkOffset3DFromInt3(dstOffsetBase),
		src:           src.base,
		srcAspects:    formatutil.Aspects(src.format),
		srcBaseLayer:  src.baseLayer,
		srcLayers:     src.layers,
		srcMipLevel:   uint8(srcMipLevel),
		srcOffset:     vkOffset3DFromInt3(srcOffsetBase),
		extent:        vkExtent3DFromInt3(extentBase),
		queueFamilies: families,
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
			SType: vk.STRUCTURE_TYPE_IMAGE_COPY_2,
			SrcSubresource: vk.ImageSubresourceLayers{
				AspectMask:     job.srcAspects,
				MipLevel:       uint32(job.srcMipLevel),
				BaseArrayLayer: job.srcBaseLayer,
				LayerCount:     job.srcLayers,
			},
			SrcOffset: job.srcOffset,
			DstSubresource: vk.ImageSubresourceLayers{
				AspectMask:     job.dstAspects,
				MipLevel:       uint32(job.dstMipLevel),
				BaseArrayLayer: job.dstBaseLayer,
				LayerCount:     job.dstLayers,
			},
			DstOffset: job.dstOffset,
			Extent:    job.extent,
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
	dst            *imageData
	dstAspects     vk.ImageAspectFlags
	dstBaseLayer   uint32
	dstLayers      uint32
	dstMipLevel    uint8
	dstOffset      vk.Offset3D
	src            UnsafePointer
	srcRowLength   uint32
	srcImageHeight uint32
	extent         vk.Extent3D
	queueFamilies  uint32
}

// Only the first mip level is copied.
//
// TODO: better comment
func EnqueueCopyMemoryToImage(
	jq *JobQueue,
	dst *Image, dstOffset [3]int,
	src Slice[byte], srcRowLength, srcImageHeight int,
	extent [3]int) {
	// TODO: validation

	dstAspects := formatutil.Aspects(dst.format)
	dstMipLevel := int(dst.baseMipLevel)
	dstOffsetBlocks := int3(dstOffset).Div(formatBlockExtent(dst.format))
	extentBlocks := divRoundUp3(extent, formatBlockExtent(dst.format))

	dstOffsetBase, extentBase := copyRectInTexels(
		dst.base.format, int3FromVkExtent3D(dst.base.extent),
		dstMipLevel,
		dstOffsetBlocks, extentBlocks)

	families := chooseQueueFamiliesForImageCopy(
		dst.base.format, int3FromVkExtent3D(dst.base.extent),
		dstAspects, dstMipLevel,
		dstOffsetBlocks, extentBlocks)

	// BUG: we need to exclude the transfer-only queue family if the src isn't
	// aligned to a 4-byte boundary.

	jq.Enqueue(&copyMemoryToImageJob{
		dst:            dst.base,
		dstAspects:     dstAspects,
		dstBaseLayer:   dst.baseLayer,
		dstLayers:      dst.layers,
		dstMipLevel:    uint8(dstMipLevel),
		dstOffset:      vkOffset3DFromInt3(dstOffsetBase),
		src:            UnsafePointer(SliceData(src)),
		srcRowLength:   uint32(srcRowLength),
		srcImageHeight: uint32(srcImageHeight),
		extent:         vkExtent3DFromInt3(extentBase),
		queueFamilies:  families,
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
			ImageSubresource: vk.ImageSubresourceLayers{
				AspectMask:     job.dstAspects,
				MipLevel:       uint32(job.dstMipLevel),
				BaseArrayLayer: job.dstBaseLayer,
				LayerCount:     job.dstLayers,
			},
			ImageOffset: job.dstOffset,
			ImageExtent: job.extent,
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
	dst            UnsafePointer
	dstRowLength   uint32
	dstImageHeight uint32
	src            *imageData
	srcAspects     vk.ImageAspectFlags
	srcBaseLayer   uint32
	srcLayers      uint32
	srcMipLevel    uint8
	srcOffset      vk.Offset3D
	extent         vk.Extent3D
	queueFamilies  uint32
}

func EnqueueCopyImageToMemory(
	jq *JobQueue,
	dst Slice[byte], dstRowLength, dstImageHeight int,
	src *Image, srcOffset [3]int,
	extent [3]int) {
	srcAspects := formatutil.Aspects(src.format)
	srcMipLevel := int(src.baseMipLevel)
	srcOffsetBlocks := int3(srcOffset).Div(formatBlockExtent(src.format))
	extentBlocks := divRoundUp3(extent, formatBlockExtent(src.format))

	srcOffsetBase, extentBase := copyRectInTexels(
		src.base.format, int3FromVkExtent3D(src.base.extent),
		srcMipLevel,
		srcOffsetBlocks, extentBlocks)

	families := chooseQueueFamiliesForImageCopy(
		src.base.format, int3FromVkExtent3D(src.base.extent),
		srcAspects, srcMipLevel,
		srcOffsetBlocks, extentBlocks)

	// BUG: we need to exclude the transfer-only queue family if the src isn't
	// aligned to a 4-byte boundary.

	jq.Enqueue(&copyImageToMemoryJob{
		dst:            UnsafePointer(SliceData(dst)),
		dstRowLength:   uint32(dstRowLength),
		dstImageHeight: uint32(dstImageHeight),
		src:            src.base,
		srcAspects:     srcAspects,
		srcBaseLayer:   src.baseLayer,
		srcLayers:      src.layers,
		srcMipLevel:    uint8(srcMipLevel),
		srcOffset:      vkOffset3DFromInt3(srcOffsetBase),
		extent:         vkExtent3DFromInt3(extentBase),
		queueFamilies:  families,
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
			ImageSubresource: vk.ImageSubresourceLayers{
				AspectMask:     job.srcAspects,
				MipLevel:       uint32(job.srcMipLevel),
				BaseArrayLayer: job.srcBaseLayer,
				LayerCount:     job.srcLayers,
			},
			ImageOffset: job.srcOffset,
			ImageExtent: job.extent,
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
	format Format, imageExtent [3]int,
	aspects vk.ImageAspectFlags, level int,
	offsetBlocks, extentBlocks [3]int) uint32 {
	levelExtent := minify3(imageExtent, level)
	levelExtentBlocks := divRoundUp3(levelExtent, formatBlockExtent(format))

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
				int3(offsetBlocks).Mod(granularity) != ([3]int{}) ||
				int3(extentBlocks).Mod(granularity) != ([3]int{}) {
				families &^= 1 << family
			}
		}
	}

	return families
}

func copyRectInTexels(
	format Format, imageExtent [3]int,
	level int,
	offsetBlocks, extentBlocks [3]int) ([3]int, [3]int) {
	blockExtent := int3FromVkExtent3D(formatutil.Describe(format).BlockExtent)

	offset := int3(offsetBlocks).Mul(blockExtent)
	extent := int3(extentBlocks).Mul(blockExtent).Min(int3(minify3(imageExtent, level)).Sub(offset))
	return offset, extent
}
