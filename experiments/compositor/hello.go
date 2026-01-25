//go:build ignore

package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"math/bits"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"
	"time"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
	"worldspawn/gpu/wsi"
	"worldspawn/internal/postprocess"
	"worldspawn/internal/sdl"
)

/*
	type output struct {
		window     *sdl.Window
		swapchain  *gpu.Swapchain
		redrawMu   sync.Mutex
		resizeCond sync.Cond
	}
*/

func readPNG(filename string) (*gpu.Image, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	hostImg, err := png.Decode(f)
	if err != nil {
		return nil, err
	}

	format := vk.Format(0)
	var pix []byte
	switch hostImg := hostImg.(type) {
	case *image.NRGBA:
		format = vk.FORMAT_R8G8B8A8_UNORM
		pix = hostImg.Pix
	default:
		panic("unsupported color model...")
	}

	pix2 := gpu.MakeSliceUncached[byte](len(pix))
	copy(pix2.Value(), pix)

	gpuImg := gpu.NewImage(
		format,
		[]int{hostImg.Bounds().Dx(), hostImg.Bounds().Dy()},
		gpu.WithUsage(vk.IMAGE_USAGE_STORAGE_BIT),
		gpu.WithUsage(vk.IMAGE_USAGE_SAMPLED_BIT))

	jq := new(gpu.JobQueue)
	gpuImg.EnqueueInit(jq)
	gpu.EnqueueCopyMemoryToImage(jq, gpuImg, nil, pix2, 0, 0, gpuImg.Extent())
	gpu.WaitForIdle(jq)

	return gpuImg, nil
}

func completeMipChain(extent []int) int {
	largestSide := 1
	for _, side := range extent {
		largestSide = max(largestSide, side)
	}
	return log2(largestSide) + 1
}

func log2(x int) int {
	return bits.LeadingZeros(0) - 1 - bits.LeadingZeros(uint(x))
}

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
	window.SetResizable(true)
	window.SetSize(1200, 1200)

	hehe, _ := readPNG("/home/nanokatze/pfp cropped smaller.png")

	resized := make(chan struct{}, 1)
	var redrawMu sync.Mutex

	var swapchain *wsi.Swapchain

	// compositionPipeline := compositor.Pipeline{
	// 	Program: []uint32{
	// 		0,
	// 	},
	// }

	t0 := time.Now()
	_ = t0

	redrawLocked := func() bool {
		jq := new(gpu.JobQueue)

		swapchainImageIndex := swapchain.Acquire()
		if swapchainImageIndex == -1 {
			return false
		}

		swapchainImage := swapchain.Image(swapchainImageIndex)
		swapchainImage.EnqueueInit(jq)

		postprocess.Bloom(jq, swapchainImage, hehe)

		// TODO: it could be nice if we could present any random image (it's
		// fine if that incurs a copy.)
		swapchain.Present(jq, swapchainImageIndex)

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

			currentExtent := [2]int{int(e.Data1), int(e.Data2)}

			swapchain = wsi.NewSwapchain(&wsi.SwapchainConfig{
				Window:     window,
				ColorSpace: vk.COLOR_SPACE_SRGB_NONLINEAR_KHR,
				Format:     vk.FORMAT_R8G8B8A8_UNORM,
				Extent:     currentExtent,
				ImageOptions: []gpu.ImageOption{
					gpu.WithUsage(vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT),
					gpu.WithUsage(vk.IMAGE_USAGE_STORAGE_BIT),
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
