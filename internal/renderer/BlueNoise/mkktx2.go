//go:build ignore

package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"

	"worldspawn/gpu"
	gpuktx2 "worldspawn/gpu/image/ktx2"
	"worldspawn/gpu/vk"
)

func main() {
	jq := new(gpu.JobQueue)

	gpuImg := gpu.NewImage(gpu.MakeImageConfig(vk.FORMAT_R16G16B16A16_UNORM, []int{256, 256}).WithLayers(8), 0)
	gpuImg.EnqueueInit(jq)

	for i := range gpuImg.Layers() {
		f, err := os.Open(fmt.Sprintf("2D/256_256/HDR_RGBA_%d.png", i))
		if err != nil {
			panic(err)
		}
		defer f.Close()

		img, err := png.Decode(f)
		if err != nil {
			panic(err)
		}

		imgNRGBA := img.(*image.NRGBA64)

		for i := 0; i < len(imgNRGBA.Pix); i += 2 {
			imgNRGBA.Pix[i+0], imgNRGBA.Pix[i+1] = imgNRGBA.Pix[i+1], imgNRGBA.Pix[i+0]
		}

		staging := gpu.MakeSliceUncached[byte](len(imgNRGBA.Pix))
		defer jq.Cleanup(func() { gpu.Free(gpu.UnsafePointer(gpu.SliceData(staging))) })

		copy(staging.Value(), imgNRGBA.Pix)

		gpu.EnqueueCopyMemoryToImage(
			jq,
			gpuImg.SubImage(gpu.SliceLayers{i, i + 1}), nil,
			staging, 0, 0,
			[]int{imgNRGBA.Rect.Max.X, imgNRGBA.Rect.Max.Y})
	}

	jq.Idle().Wait()

	f, err := os.Create("2D_256_256_HDR_RGBA.ktx2")
	if err != nil {
		log.Fatal(err)
	}
	if err := gpuktx2.Encode(f, gpuImg); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}
