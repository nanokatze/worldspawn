//go:build ignore

package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	_ "net/http/pprof"
	"sync"

	"worldspawn/experiments/compositor"
	"worldspawn/gpu"
	"worldspawn/gpu/rendering"
	"worldspawn/gpu/vk"
	"worldspawn/sdl"
)

/*
	type output struct {
		window     *sdl.Window
		swapchain  *gpu.Swapchain
		redrawMu   sync.Mutex
		resizeCond sync.Cond
	}
*/

func main() {
	go func() {
		log.Println(http.ListenAndServe("[::]:6060", nil))
	}()

	if err := sdl.InitSubSystem(sdl.INIT_VIDEO); err != nil {
		panic(fmt.Sprintf("failed to initialize SDL video subsystem: %v", err))
	}

	window, err := sdl.CreateWindow(
		sdl.WithBooleanProperty(sdl.PROP_WINDOW_CREATE_VULKAN_BOOLEAN, true),
		sdl.WithBooleanProperty(sdl.PROP_WINDOW_CREATE_HIGH_PIXEL_DENSITY_BOOLEAN, true))
	if err != nil {
		log.Fatal(err)
	}

	window.SetTitle("compositor")
	window.SetSize(1280, 1080)

	resized := make(chan struct{}, 1)
	var redrawMu sync.Mutex

	var swapchain *gpu.Swapchain

	compositionPipeline := compositor.Pipeline{
		Program: []uint32{
			0,
		},
	}

	redrawLocked := func() bool {
		var jq gpu.JobQueue

		swapchainImageIndex := swapchain.Acquire()
		if swapchainImageIndex == -1 {
			return false
		}
		swapchainImage := swapchain.Image(swapchainImageIndex)

		swapchainImage.EnqueueInit(&jq)

		compositionPipeline.Run(&jq)

		func() {
			rp := rendering.Begin(&jq,
				&rendering.Config{
					ColorAttachments: []rendering.Attachment{
						{
							Image:   swapchainImage,
							LoadOp:  vk.ATTACHMENT_LOAD_OP_CLEAR,
							StoreOp: vk.ATTACHMENT_STORE_OP_STORE,
							ClearValue: [4]uint32{
								math.Float32bits(0.1),
								math.Float32bits(0),
								math.Float32bits(0),
								math.Float32bits(1),
							},
						},
					},
					RenderArea: vk.Rect2D{Extent: vk.Extent2D{Width: uint32(swapchainImage.Extent()[0]), Height: uint32(swapchainImage.Extent()[1])}},
					LayerCount: 1,
				})
			defer rp.End()
		}()

		// jq.WaitForIdle()

		swapchain.Present(&jq, swapchainImageIndex)

		return true
	}

	redraw := func() bool {
		redrawMu.Lock()
		defer redrawMu.Unlock()

		return redrawLocked()
	}

	go func() {
		for {
			<-resized
			for redraw() {
			}
		}
	}()

eventLoop:
	for {
		e, err := sdl.WaitEvent()
		if err != nil {
			log.Fatalln("WaitEvent failed", err)
		}

		switch e := e.(type) {
		case *sdl.QuitEvent:
			break eventLoop

		case *sdl.WindowPixelSizeChangedEvent:
			redrawMu.Lock()

			currentExtent := [3]int{int(e.Data1), int(e.Data2), 1}

			swapchain = gpu.NewSwapchain(&gpu.SwapchainConfig{
				Window:     window,
				ColorSpace: vk.COLOR_SPACE_SRGB_NONLINEAR_KHR,
				ImageConfig: &gpu.ImageConfig{
					Dim:       gpu.ImageDim2D,
					Extent:    currentExtent,
					Layers:    1,
					MipLevels: 1,
					Samples:   1,
					Format:    vk.FORMAT_R8G8B8A8_SRGB,
					Usage:     gpu.ImageUsageAttachment,
				},
				OldSwapchain: swapchain,
			})

			// Redraw a single frame at this size.
			redrawLocked()

			select {
			case resized <- struct{}{}:
			default:
			}

			redrawMu.Unlock()

		default:
			_ = e
		}
	}
}
