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

type copyImageJob struct {
	dst           *imageData
	dstBounds     imageBounds
	dstOffset     vk.Offset3D
	src           *imageData
	srcBounds     imageBounds
	srcOffset     vk.Offset3D
	extent        vk.Extent3D
	queueFamilies uint32
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
	dst *Image, dstOffset point,
	src *Image, srcOffset point,
	extent point) {
	extentBlocks := divByBlockExtentRoundUp(extent, src.format)

	dstOffsetBlocks := divByBlockExtent(dstOffset, dst.format)
	dstOffsetBase, _ := blockRectToTexelRect(dst.data, dst.bounds, dstOffsetBlocks, extentBlocks)
	dstQueueFamilies := copyCapableQueueFamilies(dst.data, dst.bounds)

	srcOffsetBlocks := divByBlockExtent(srcOffset, src.format)
	srcOffsetBase, extentBase := blockRectToTexelRect(src.data, src.bounds, srcOffsetBlocks, extentBlocks)
	srcQueueFamilies := copyCapableQueueFamilies(src.data, src.bounds)

	queueFamilies := dstQueueFamilies & srcQueueFamilies

	jq.Enqueue(&copyImageJob{
		dst:           dst.data,
		dstBounds:     dst.bounds,
		dstOffset:     vkOffset3DFromInt3(dstOffsetBase),
		src:           src.data,
		srcBounds:     src.bounds,
		srcOffset:     vkOffset3DFromInt3(srcOffsetBase),
		extent:        vkExtent3DFromInt3(extentBase),
		queueFamilies: queueFamilies,
	})
}

func (job *copyImageJob) Info() JobInfo {
	return JobInfo{
		QueueFamilies: job.queueFamilies,
	}
}

func (job *copyImageJob) Exec(q *DeviceQueue) {
	q.Commands(func(cb vk.CommandBuffer) {
		var pinner runtime.Pinner
		defer pinner.Unpin()

		region := &vk.ImageCopy2{
			SType:          vk.STRUCTURE_TYPE_IMAGE_COPY_2,
			SrcSubresource: job.srcBounds.VkImageSubresourceLayers(job.src.config.format),
			SrcOffset:      job.srcOffset,
			DstSubresource: job.dstBounds.VkImageSubresourceLayers(job.dst.config.format),
			DstOffset:      job.dstOffset,
			Extent:         job.extent,
		}
		pinner.Pin(region)

		VkFns.CmdCopyImage2(cb, &vk.CopyImageInfo2{
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
	dstBounds      imageBounds
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
	dst *Image, dstOffset []int,
	src Slice[byte], srcRowLength, srcImageHeight int,
	extent []int) {
	enqueueCopyMemoryToImage(jq, dst, offset3(dstOffset), src, srcRowLength, srcImageHeight, extent3(extent))
}

func enqueueCopyMemoryToImage(
	jq *JobQueue,
	dst *Image, dstOffset point,
	src Slice[byte], srcRowLength, srcImageHeight int,
	extent point) {
	// TODO: validation

	extentBlocks := divByBlockExtentRoundUp(extent, dst.format)

	dstOffsetBlocks := divByBlockExtent(dstOffset, dst.format)
	dstOffsetBase, extentBase := blockRectToTexelRect(dst.data, dst.bounds, dstOffsetBlocks, extentBlocks)

	queueFamilies := copyCapableQueueFamilies(dst.data, dst.bounds)

	// BUG: we need to exclude the transfer-only queue family if the src isn't
	// aligned to a 4-byte boundary.

	jq.Enqueue(&copyMemoryToImageJob{
		dst:            dst.data,
		dstBounds:      dst.bounds,
		dstOffset:      vkOffset3DFromInt3(dstOffsetBase),
		src:            UnsafePointer(SliceData(src)),
		srcRowLength:   uint32(srcRowLength),
		srcImageHeight: uint32(srcImageHeight),
		extent:         vkExtent3DFromInt3(extentBase),
		queueFamilies:  queueFamilies,
	})
}

func (job *copyMemoryToImageJob) Info() JobInfo {
	return JobInfo{
		QueueFamilies: job.queueFamilies,
	}
}

func (job *copyMemoryToImageJob) Exec(q *DeviceQueue) {
	q.Commands(func(cb vk.CommandBuffer) {
		var pinner runtime.Pinner
		defer pinner.Unpin()

		region := &vk.DeviceMemoryImageCopyKHR{
			SType: vk.STRUCTURE_TYPE_DEVICE_MEMORY_IMAGE_COPY_KHR,
			AddressRange: vk.DeviceAddressRangeKHR{
				Address: vk.DeviceAddress(job.src),
				Size:    ^vk.DeviceSize(0), // BUG: provide a real size
			},
			AddressFlags: vk.AddressCommandFlagsKHR(vk.ADDRESS_COMMAND_FULLY_BOUND_BIT_KHR) |
				vk.AddressCommandFlagsKHR(vk.ADDRESS_COMMAND_UNKNOWN_STORAGE_BUFFER_USAGE_BIT_KHR),
			AddressRowLength:   uint32(job.srcRowLength),
			AddressImageHeight: uint32(job.srcImageHeight),
			ImageSubresource:   job.dstBounds.VkImageSubresourceLayers(job.dst.config.format),
			ImageOffset:        job.dstOffset,
			ImageExtent:        job.extent,
		}
		pinner.Pin(region)

		VkFns.CmdCopyMemoryToImageKHR(cb, &vk.CopyDeviceMemoryImageInfoKHR{
			SType:       vk.STRUCTURE_TYPE_COPY_DEVICE_MEMORY_IMAGE_INFO_KHR,
			Image:       job.dst.vkImage,
			RegionCount: 1,
			PRegions:    region,
		})
	})
}

type copyImageToMemoryJob struct {
	dst            UnsafePointer
	dstRowLength   uint32
	dstImageHeight uint32
	src            *imageData
	srcBounds      imageBounds
	srcOffset      vk.Offset3D
	extent         vk.Extent3D
	queueFamilies  uint32
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
	src *Image, srcOffset point,
	extent point) {
	extentBlocks := divByBlockExtentRoundUp(extent, src.format)

	srcOffsetBlocks := divByBlockExtent(srcOffset, src.format)
	srcOffsetBase, extentBase := blockRectToTexelRect(src.data, src.bounds, srcOffsetBlocks, extentBlocks)
	queueFamilies := copyCapableQueueFamilies(src.data, src.bounds)

	// BUG: we need to exclude the transfer-only queue family if the src isn't
	// aligned to a 4-byte boundary.

	jq.Enqueue(&copyImageToMemoryJob{
		dst:            UnsafePointer(SliceData(dst)),
		dstRowLength:   uint32(dstRowLength),
		dstImageHeight: uint32(dstImageHeight),
		src:            src.data,
		srcBounds:      src.bounds,
		srcOffset:      vkOffset3DFromInt3(srcOffsetBase),
		extent:         vkExtent3DFromInt3(extentBase),
		queueFamilies:  queueFamilies,
	})
}

func (job *copyImageToMemoryJob) Info() JobInfo {
	return JobInfo{
		QueueFamilies: job.queueFamilies,
	}
}

func (job *copyImageToMemoryJob) Exec(q *DeviceQueue) {
	q.Commands(func(cb vk.CommandBuffer) {
		var pinner runtime.Pinner
		defer pinner.Unpin()

		region := &vk.DeviceMemoryImageCopyKHR{
			SType: vk.STRUCTURE_TYPE_DEVICE_MEMORY_IMAGE_COPY_KHR,
			AddressRange: vk.DeviceAddressRangeKHR{
				Address: vk.DeviceAddress(job.dst),
				Size:    ^vk.DeviceSize(0), // BUG: provide a real size
			},
			AddressFlags: vk.AddressCommandFlagsKHR(vk.ADDRESS_COMMAND_FULLY_BOUND_BIT_KHR) |
				vk.AddressCommandFlagsKHR(vk.ADDRESS_COMMAND_UNKNOWN_STORAGE_BUFFER_USAGE_BIT_KHR),
			AddressRowLength:   uint32(job.dstRowLength),
			AddressImageHeight: uint32(job.dstImageHeight),
			ImageSubresource:   job.srcBounds.VkImageSubresourceLayers(job.src.config.format),
			ImageOffset:        job.srcOffset,
			ImageExtent:        job.extent,
		}
		pinner.Pin(region)

		VkFns.CmdCopyImageToMemoryKHR(cb, &vk.CopyDeviceMemoryImageInfoKHR{
			SType:       vk.STRUCTURE_TYPE_COPY_DEVICE_MEMORY_IMAGE_INFO_KHR,
			Image:       job.src.vkImage,
			RegionCount: 1,
			PRegions:    region,
		})
	})
}

// Queue families that are capable of copying to or from the image.
func copyCapableQueueFamilies(data *imageData, bounds imageBounds) QueueFamilyMask {
	families := Topology.QueueFamilies(vk.QueueFlags(vk.QUEUE_TRANSFER_BIT))

	// TODO: determine this differently
	aspects := bounds.VkImageSubresourceRange(data.config.format).AspectMask
	if aspects&vk.ImageAspectFlags(vk.IMAGE_ASPECT_DEPTH_BIT) != 0 {
		families &= Topology.QueueFamilies(vk.QueueFlags(vk.QUEUE_GRAPHICS_BIT))
	}
	if aspects&vk.ImageAspectFlags(vk.IMAGE_ASPECT_STENCIL_BIT) != 0 {
		families &= Topology.QueueFamilies(vk.QueueFlags(vk.QUEUE_GRAPHICS_BIT))
	}

	return families
}

func blockRectToTexelRect(
	data *imageData, bounds imageBounds,
	offsetBlocks, extentBlocks point) (point, point) {
	blockExtent := formatutil.Describe(data.config.format).BlockExtent
	levelExtent := minify3(data.config.extent, bounds.FirstMip())

	offset := offsetBlocks.Mul(blockExtent)
	extent := min3(extentBlocks.Mul(blockExtent), levelExtent.Sub(offset))
	return offset, extent
}

// TODO: optimize for block sides? We don't need the general division here.

func divByBlockExtent(x point, yFormat vk.Format) point {
	y := formatutil.Describe(yFormat).BlockExtent
	return point{
		x[0] / y[0],
		x[1] / y[1],
		x[2] / y[2],
	}
}

func divByBlockExtentRoundUp(x point, yFormat vk.Format) point {
	y := formatutil.Describe(yFormat).BlockExtent
	return point{
		(x[0] + y[0] - 1) / y[0],
		(x[1] + y[1] - 1) / y[1],
		(x[2] + y[2] - 1) / y[2],
	}
}
