package rendering

import (
	"runtime"
	"sync"
	"unsafe"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

type RenderingConfig struct {
	ColorAttachments  []Attachment
	DepthAttachment   *Attachment
	StencilAttachment *Attachment

	RenderArea vk.Rect2D
	LayerCount uint32
}

// TODO: rename to RenderingAttachment?
type Attachment struct {
	Image *gpu.Image

	LoadOp  vk.AttachmentLoadOp
	StoreOp vk.AttachmentStoreOp
	// When LoadOp is CLEAR, ClearValue specifies the bit pattern that the
	// samples in this attachment will be set to at the beginning of the render
	// pass.
	ClearValue [4]uint32
}

type RenderPass struct {
	cb          vk.CommandBuffer
	garbage     []func()
	queueFamily int

	jq *gpu.JobQueue
}

var device vk.Device

var vkFns vk.DeviceFuncs

var initOnce sync.Once

func initDevice() {
	initOnce.Do(func() {
		device = gpu.Device()
		vkFns.Init(device)
	})
}

// TODO: think of a variant that supports concurrent recording.
func BeginRendering(jq *gpu.JobQueue, config *RenderingConfig) *RenderPass {
	initDevice()

	var pinner runtime.Pinner
	defer pinner.Unpin()

	rp := new(RenderPass)
	rp.jq = jq

	// TODO: stop using the deprecated MinimumCapable
	queueFamily := gpu.BestQueueFamily(0b001)
	rp.queueFamily = queueFamily

	cb := cbcaches[queueFamily].Get()
	rp.cb = cb.Vk()
	rp.Cleanup(func() { cbcaches[queueFamily].Put(cb) })

	must(vkFns.BeginCommandBuffer(rp.cb, &vk.CommandBufferBeginInfo{
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
		rp.garbage = append(rp.garbage, func() { vkFns.DestroyImageView(device, vkImageView, nil) })
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

		rp.garbage = append(rp.garbage, func() { vkFns.DestroyImageView(device, vkImageView, nil) })
	}

	vkFns.CmdBeginRendering(rp.cb, renderingInfo)

	gpu.BindDescriptorSet(rp.cb, vk.PIPELINE_BIND_POINT_GRAPHICS)

	// Graphics state we don't map from our abstraction goes here

	vkFns.CmdSetVertexInputEXT(rp.cb, 0, nil, 0, nil)

	return rp
}

// TODO: remove when we can use ms only
func (rp *RenderPass) SetIndexBuffer(indexType vk.IndexType, indexBuffer gpu.UnsafePointer) {
	if indexBuffer != vk.NULL_HANDLE {
		buffer, offset := gpu.BufferAndOffset(indexBuffer)
		vkFns.CmdBindIndexBuffer(rp.cb, buffer, offset, indexType)
	} else {
		// needs maintenance6: vkProcs.CmdBindIndexBuffer(rp.cb_, vk.NULL_HANDLE, 0, 0)
	}
}

func (rp *RenderPass) SetPrimitiveTopology(primitiveTopology vk.PrimitiveTopology) {
	vkFns.CmdSetPrimitiveTopologyEXT(rp.cb, primitiveTopology)
}

func (rp *RenderPass) SetPrimitiveRestartEnable(primitiveRestartEnable bool) {
	vkFns.CmdSetPrimitiveRestartEnableEXT(rp.cb, vkBool32(primitiveRestartEnable))
}

func (rp *RenderPass) SetViewports(viewports []vk.Viewport) {
	vkFns.CmdSetViewportWithCountEXT(rp.cb, uint32(len(viewports)), unsafe.SliceData(viewports))
}

func (rp *RenderPass) SetScissors(scissors []vk.Rect2D) {
	vkFns.CmdSetScissorWithCountEXT(rp.cb, uint32(len(scissors)), unsafe.SliceData(scissors))
}

func (rp *RenderPass) SetRasterizerDiscardEnable(rasterizerDiscardEnable bool) {
	vkFns.CmdSetRasterizerDiscardEnableEXT(rp.cb, vkBool32(rasterizerDiscardEnable))
}

func (rp *RenderPass) SetPolygonMode(polygonMode vk.PolygonMode) {
	vkFns.CmdSetPolygonModeEXT(rp.cb, polygonMode)
}

func (rp *RenderPass) SetCullMode(cullMode vk.CullModeFlags) {
	vkFns.CmdSetCullModeEXT(rp.cb, cullMode)
}

func (rp *RenderPass) SetFrontFace(frontFace vk.FrontFace) {
	vkFns.CmdSetFrontFaceEXT(rp.cb, frontFace)
}

func (rp *RenderPass) SetDepthBiasEnable(depthBiasEnable bool) {
	vkFns.CmdSetDepthBiasEnableEXT(rp.cb, vkBool32(depthBiasEnable))
}

func (rp *RenderPass) SetRasterizationSamples(samples int) {
	vkFns.CmdSetRasterizationSamplesEXT(rp.cb, vk.SampleCountFlagBits(samples))
}

func (rp *RenderPass) SetSampleMask(sampleMask uint32) {
	vkFns.CmdSetSampleMaskEXT(rp.cb, 1, (*vk.SampleMask)(&sampleMask))
}

func (rp *RenderPass) SetAlphaToCoverageEnable(alphaToCoverageEnable bool) {
	vkFns.CmdSetAlphaToCoverageEnableEXT(rp.cb, vkBool32(alphaToCoverageEnable))
}

func (rp *RenderPass) SetDepthTestEnable(depthTestEnable bool) {
	vkFns.CmdSetDepthTestEnableEXT(rp.cb, vkBool32(depthTestEnable))
}

func (rp *RenderPass) SetDepthWriteEnable(depthWriteEnable bool) {
	vkFns.CmdSetDepthWriteEnableEXT(rp.cb, vkBool32(depthWriteEnable))
}

func (rp *RenderPass) SetDepthCompareOp(depthCompareOp vk.CompareOp) {
	vkFns.CmdSetDepthCompareOpEXT(rp.cb, depthCompareOp)
}

func (rp *RenderPass) SetDepthBoundsTestEnable(depthBoundsTestEnable bool) {
	vkFns.CmdSetDepthBoundsTestEnableEXT(rp.cb, vkBool32(depthBoundsTestEnable))
}

func (rp *RenderPass) SetStencilTestEnable(stencilTestEnable bool) {
	vkFns.CmdSetStencilTestEnableEXT(rp.cb, vkBool32(stencilTestEnable))
}

func (rp *RenderPass) SetColorBlendEnable(attachment int, colorBlendEnable bool) {
	tmp := vkBool32(colorBlendEnable)
	vkFns.CmdSetColorBlendEnableEXT(rp.cb, uint32(attachment), 1, &tmp)
}

func (rp *RenderPass) SetColorBlendEquation(attachment int, colorBlendEquation vk.ColorBlendEquationEXT) {
	vkFns.CmdSetColorBlendEquationEXT(rp.cb, uint32(attachment), 1, &colorBlendEquation)
}

func (rp *RenderPass) SetColorWriteMask(attachment int, colorWriteMask uint32) {
	vkFns.CmdSetColorWriteMaskEXT(rp.cb, uint32(attachment), 1, (*vk.ColorComponentFlags)(&colorWriteMask))
}

func (rp *RenderPass) SetBlendConstants(blendConstants [4]float32) {
	panic("not implemented")
	// vkFns.CmdSetBlendConstants(rp.cb, &blendConstants)
}

func (rp *RenderPass) SetShader(stage vk.ShaderStageFlagBits, shader *Func) {
	vkShader := vk.ShaderEXT(vk.NULL_HANDLE)
	if shader != nil {
		vkShader = shader.vkShader()
	}
	vkFns.CmdBindShadersEXT(rp.cb, 1, &stage, &vkShader)
}

func (rp *RenderPass) SetShaderArgs(p any) {
	gpu.PushConstants(rp.cb, asbytes(p))
}

func (rp *RenderPass) Draw(vertexCount uint32, instanceCount uint32, firstVertex uint32, firstInstance uint32) {
	vkFns.CmdDraw(rp.cb, vertexCount, instanceCount, firstVertex, firstInstance)
}

func (rp *RenderPass) DrawIndexed(indexCount uint32, instanceCount uint32, firstIndex uint32, vertexOffset int32, firstInstance uint32) {
	vkFns.CmdDrawIndexed(rp.cb, indexCount, instanceCount, firstIndex, vertexOffset, firstInstance)
}

// TODO: renderpass barrier

func (rp *RenderPass) Cleanup(f func()) {
	rp.garbage = append(rp.garbage, f)
}

func (rp *RenderPass) End() {
	vkFns.CmdEndRendering(rp.cb)

	rp.jq.Enqueue(&renderPassJob{
		cb:          rp.cb,
		garbage:     rp.garbage,
		queueFamily: int(rp.queueFamily),
	})

	// Zero out to help diagnose misuse
	*rp = RenderPass{}
}

type renderPassJob struct {
	cb      vk.CommandBuffer
	garbage []func()

	queueFamily int
}

func (job *renderPassJob) Info() gpu.JobInfo {
	return gpu.JobInfo{
		QueueFamilies: 1 << job.queueFamily,
	}
}

func (job *renderPassJob) Exec(q *gpu.CommandQueue) {
	q.CommandBuffer(job.cb)

	q.Cleanup(func() {
		for _, g := range job.garbage {
			g()
		}
	})
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
	vkImage, vkImageSubresourceRange := img.Vk()

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
