package draw

import (
	"runtime"
	"sync"
	"unsafe"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

var device vk.Device

var vkFns vk.DeviceFuncs

var initOnce sync.Once

func initDevice() {
	initOnce.Do(func() {
		device = gpu.Device()
		vkFns.Init(device)
	})
}

type Config struct {
	ColorAttachments  []Attachment
	DepthAttachment   *Attachment
	StencilAttachment *Attachment

	// TODO: make these use ints instead of vk structs

	RenderArea vk.Rect2D
	LayerCount uint32
}

type Attachment struct {
	Image *gpu.Image

	// TODO: can we fold LoadOp + ClearValue in one?
	LoadOp  vk.AttachmentLoadOp
	StoreOp vk.AttachmentStoreOp
	// When LoadOp is CLEAR, ClearValue specifies the bit pattern that the
	// samples in this attachment will be set to at the beginning of the pass.
	ClearValue [4]uint32
}

type Pass struct {
	cb          vk.CommandBuffer
	garbage     []func()
	queueFamily int

	jq *gpu.JobQueue
}

// TODO: think of a variant that supports concurrent recording.
func Begin(jq *gpu.JobQueue, config *Config) *Pass {
	initDevice()

	var pinner runtime.Pinner
	defer pinner.Unpin()

	pass := new(Pass)
	pass.jq = jq

	// TODO: get a mask of GRAPHICS-capable queue families, the permutation and
	// find the lsb. We could also just find the first set bit because it's
	queueFamily := gpu.BestQueueFamily(vk.QueueFlags(vk.QUEUE_GRAPHICS_BIT))
	pass.queueFamily = queueFamily

	cb := cbcaches[queueFamily].Get()
	pass.cb = cb.Vk()
	pass.Cleanup(func() { cbcaches[queueFamily].Put(cb) })

	must(vkFns.BeginCommandBuffer(pass.cb, &vk.CommandBufferBeginInfo{
		SType: vk.STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
		Flags: vk.CommandBufferUsageFlags(vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT),
	}))

	renderingInfo := &vk.RenderingInfo{
		SType:      vk.STRUCTURE_TYPE_RENDERING_INFO,
		RenderArea: config.RenderArea,
		LayerCount: config.LayerCount,
	}

	colorAttachments := make([]vk.RenderingAttachmentInfo, len(config.ColorAttachments))
	for i, a := range config.ColorAttachments {
		vkImageView := newRenderingVkImageView(a.Image, vk.ImageUsageFlags(vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT))

		colorAttachments[i] = vk.RenderingAttachmentInfo{
			SType:       vk.STRUCTURE_TYPE_RENDERING_ATTACHMENT_INFO,
			ImageView:   vkImageView,
			ImageLayout: vk.IMAGE_LAYOUT_GENERAL,
			LoadOp:      a.LoadOp,
			StoreOp:     a.StoreOp,
			ClearValue:  a.ClearValue,
		}

		// TODO: have a single function for all of the temporary image views
		// here
		pass.garbage = append(pass.garbage, func() { vkFns.DestroyImageView(device, vkImageView, nil) })
	}
	renderingInfo.ColorAttachmentCount = uint32(len(colorAttachments))
	renderingInfo.PColorAttachments = pinnedSliceData(&pinner, colorAttachments)

	if config.DepthAttachment != nil {
		attachment := config.DepthAttachment

		vkImageView := newRenderingVkImageView(attachment.Image, vk.ImageUsageFlags(vk.IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT))

		renderingInfo.PDepthAttachment = pinned(&pinner, &vk.RenderingAttachmentInfo{
			SType:       vk.STRUCTURE_TYPE_RENDERING_ATTACHMENT_INFO,
			ImageView:   vkImageView,
			ImageLayout: vk.IMAGE_LAYOUT_GENERAL,
			LoadOp:      attachment.LoadOp,
			StoreOp:     attachment.StoreOp,
			ClearValue:  attachment.ClearValue,
		})

		pass.garbage = append(pass.garbage, func() { vkFns.DestroyImageView(device, vkImageView, nil) })
	}

	vkFns.CmdBeginRendering(pass.cb, renderingInfo)

	gpu.BindDescriptorSet(pass.cb, vk.PIPELINE_BIND_POINT_GRAPHICS)

	// Graphics state we don't map from our abstraction goes here

	vkFns.CmdSetVertexInputEXT(pass.cb, 0, nil, 0, nil)

	return pass
}

// TODO: remove when we can use ms only
func (pass *Pass) SetIndexBuffer(indexType vk.IndexType, indexBuffer gpu.UnsafePointer) {
	if indexBuffer != vk.NULL_HANDLE {
		vkFns.CmdBindIndexBuffer3KHR(pass.cb,
			&vk.BindIndexBuffer3InfoKHR{
				SType: vk.STRUCTURE_TYPE_BIND_INDEX_BUFFER_3_INFO_KHR,
				AddressRange: vk.DeviceAddressRangeKHR{
					Address: vk.DeviceAddress(indexBuffer),
					Size:    ^vk.DeviceSize(0), // provide a real address
				},
				AddressFlags: vk.AddressCommandFlagsKHR(vk.ADDRESS_COMMAND_FULLY_BOUND_BIT_KHR) |
					vk.AddressCommandFlagsKHR(vk.ADDRESS_COMMAND_UNKNOWN_STORAGE_BUFFER_USAGE_BIT_KHR),
				IndexType: indexType,
			})
	} else {
		vkFns.CmdBindIndexBuffer3KHR(pass.cb,
			&vk.BindIndexBuffer3InfoKHR{
				SType: vk.STRUCTURE_TYPE_BIND_INDEX_BUFFER_3_INFO_KHR,
			})
	}
}

func (pass *Pass) SetPrimitiveTopology(primitiveTopology vk.PrimitiveTopology) {
	vkFns.CmdSetPrimitiveTopologyEXT(pass.cb, primitiveTopology)
}

func (pass *Pass) SetPrimitiveRestartEnable(primitiveRestartEnable bool) {
	vkFns.CmdSetPrimitiveRestartEnableEXT(pass.cb, vkBool32(primitiveRestartEnable))
}

func (pass *Pass) SetViewports(viewports []vk.Viewport) {
	vkFns.CmdSetViewportWithCountEXT(pass.cb, uint32(len(viewports)), unsafe.SliceData(viewports))
}

func (pass *Pass) SetScissors(scissors []vk.Rect2D) {
	vkFns.CmdSetScissorWithCountEXT(pass.cb, uint32(len(scissors)), unsafe.SliceData(scissors))
}

func (pass *Pass) SetRasterizerDiscardEnable(rasterizerDiscardEnable bool) {
	vkFns.CmdSetRasterizerDiscardEnableEXT(pass.cb, vkBool32(rasterizerDiscardEnable))
}

func (pass *Pass) SetPolygonMode(polygonMode vk.PolygonMode) {
	vkFns.CmdSetPolygonModeEXT(pass.cb, polygonMode)
}

func (pass *Pass) SetCullMode(cullMode vk.CullModeFlags) {
	vkFns.CmdSetCullModeEXT(pass.cb, cullMode)
}

func (pass *Pass) SetFrontFace(frontFace vk.FrontFace) {
	vkFns.CmdSetFrontFaceEXT(pass.cb, frontFace)
}

func (pass *Pass) SetDepthBiasEnable(depthBiasEnable bool) {
	vkFns.CmdSetDepthBiasEnableEXT(pass.cb, vkBool32(depthBiasEnable))
}

func (pass *Pass) SetRasterizationSamples(samples int) {
	vkFns.CmdSetRasterizationSamplesEXT(pass.cb, vk.SampleCountFlagBits(samples))
}

func (pass *Pass) SetSampleMask(sampleMask uint32) {
	vkFns.CmdSetSampleMaskEXT(pass.cb, 1, (*vk.SampleMask)(&sampleMask))
}

func (pass *Pass) SetAlphaToCoverageEnable(alphaToCoverageEnable bool) {
	vkFns.CmdSetAlphaToCoverageEnableEXT(pass.cb, vkBool32(alphaToCoverageEnable))
}

func (pass *Pass) SetDepthTestEnable(depthTestEnable bool) {
	vkFns.CmdSetDepthTestEnableEXT(pass.cb, vkBool32(depthTestEnable))
}

func (pass *Pass) SetDepthWriteEnable(depthWriteEnable bool) {
	vkFns.CmdSetDepthWriteEnableEXT(pass.cb, vkBool32(depthWriteEnable))
}

func (pass *Pass) SetDepthCompareOp(depthCompareOp vk.CompareOp) {
	vkFns.CmdSetDepthCompareOpEXT(pass.cb, depthCompareOp)
}

func (pass *Pass) SetDepthBoundsTestEnable(depthBoundsTestEnable bool) {
	vkFns.CmdSetDepthBoundsTestEnableEXT(pass.cb, vkBool32(depthBoundsTestEnable))
}

func (pass *Pass) SetStencilTestEnable(stencilTestEnable bool) {
	vkFns.CmdSetStencilTestEnableEXT(pass.cb, vkBool32(stencilTestEnable))
}

func (pass *Pass) SetColorBlendEnable(attachmentIndex int, colorBlendEnable bool) {
	tmp := vkBool32(colorBlendEnable)
	vkFns.CmdSetColorBlendEnableEXT(pass.cb, uint32(attachmentIndex), 1, &tmp)
}

func (pass *Pass) SetColorBlendEquation(attachmentIndex int, colorBlendEquation vk.ColorBlendEquationEXT) {
	vkFns.CmdSetColorBlendEquationEXT(pass.cb, uint32(attachmentIndex), 1, &colorBlendEquation)
}

func (pass *Pass) SetColorWriteMask(attachmentIndex int, colorWriteMask uint32) {
	vkFns.CmdSetColorWriteMaskEXT(pass.cb, uint32(attachmentIndex), 1, (*vk.ColorComponentFlags)(&colorWriteMask))
}

func (pass *Pass) SetBlendConstants(blendConstants [4]float32) {
	panic("not implemented")
	// vkFns.CmdSetBlendConstants(rp.cb, &blendConstants)
}

// TODO: these should take stronger-typed Shader things
func (pass *Pass) SetVertexShader(shader *Shader) {
	pass.setShader(vk.SHADER_STAGE_VERTEX_BIT, shader)
}

func (pass *Pass) SetFragmentShader(shader *Shader) {
	pass.setShader(vk.SHADER_STAGE_FRAGMENT_BIT, shader)
}

func (pass *Pass) setShader(stage vk.ShaderStageFlagBits, shader *Shader) {
	vkShader := shader.vkShader()
	vkFns.CmdBindShadersEXT(pass.cb, 1, &stage, &vkShader)
}

func (pass *Pass) SetShaderArgs(p any) {
	gpu.PushConstants(pass.cb, asbytes(p))
}

func (pass *Pass) Draw(vertexCount uint32, instanceCount uint32, firstVertex uint32, firstInstance uint32) {
	vkFns.CmdDraw(pass.cb, vertexCount, instanceCount, firstVertex, firstInstance)
}

func (pass *Pass) DrawIndexed(indexCount uint32, instanceCount uint32, firstIndex uint32, vertexOffset int32, firstInstance uint32) {
	vkFns.CmdDrawIndexed(pass.cb, indexCount, instanceCount, firstIndex, vertexOffset, firstInstance)
}

// TODO: renderpass barrier

func (pass *Pass) Cleanup(f func()) {
	pass.garbage = append(pass.garbage, f)
}

func (pass *Pass) End() {
	vkFns.CmdEndRendering(pass.cb)

	pass.jq.Enqueue(&job{
		cb:          pass.cb,
		garbage:     pass.garbage,
		queueFamily: int(pass.queueFamily),
	})

	// Zero out to help diagnose misuse
	*pass = Pass{}
}

func vkImageViewType(dim gpu.ImageDim) vk.ImageViewType {
	switch dim {
	case gpu.ImageDim1D:
		return vk.IMAGE_VIEW_TYPE_1D_ARRAY
	case gpu.ImageDim2D:
		return vk.IMAGE_VIEW_TYPE_2D_ARRAY
	case gpu.ImageDimCube:
		return vk.IMAGE_VIEW_TYPE_CUBE_ARRAY
	case gpu.ImageDim3D:
		return vk.IMAGE_VIEW_TYPE_3D
	default:
		panic("unreachable")
	}
}

func newRenderingVkImageView(img *gpu.Image, usage vk.ImageUsageFlags) vk.ImageView {
	vkImage, vkImageSubresourceRange := img.VkImage()

	var vkImageView vk.ImageView
	must(vkFns.CreateImageView(device, &vk.ImageViewCreateInfo{
		SType: vk.STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
		PNext: unsafe.Pointer(&vk.ImageViewUsageCreateInfo{
			SType: vk.STRUCTURE_TYPE_IMAGE_VIEW_USAGE_CREATE_INFO,
			Usage: usage,
		}),
		Image:            vkImage,
		ViewType:         vkImageViewType(img.Dim()), // TODO: I think only VK_IMAGE_VIEW_TYPE_2D_ARRAY is possible?
		Format:           img.Format(),
		SubresourceRange: vkImageSubresourceRange,
	}, nil, &vkImageView))
	return vkImageView
}

// TODO: move elsewhere
func vkBool32(x bool) vk.Bool32 {
	if x {
		return vk.TRUE
	} else {
		return vk.FALSE
	}
}
