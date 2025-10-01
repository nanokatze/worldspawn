package main

import (
	"io"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
	"worldspawn/image/ktx2"
	"worldspawn/internal/renderer"
)

// TODO: rename this file to something else

var texturecache = make(map[string]*renderer.Texture)
var modelcache = make(map[game.GeometryPacked]*renderer.Mesh)

// TODO: should support streaming etc.
func texture(filename string) *renderer.Texture {
	t, ok := texturecache[filename]
	if !ok {
		// TODO: move this code into its own func + handle errors and everything.

		f, err := game.Data.Open(filename)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		textureHeader, err := ktx2.NewFile(f)
		if err != nil {
			panic(err)
		}

		viewType := vk.IMAGE_VIEW_TYPE_2D
		if textureHeader.FaceCount == 6 {
			viewType = vk.IMAGE_VIEW_TYPE_CUBE
		}

		layers := int(textureHeader.FaceCount) * int(max(textureHeader.LayerCount, 1))

		t = renderer.NewTexture(
			viewType,
			[3]int{
				int(textureHeader.Width),
				int(max(textureHeader.Height, 1)),
				int(max(textureHeader.Depth, 1)),
			},
			len(textureHeader.MipLevels),
			layers,
			vk.Format(textureHeader.VkFormat))

		var wg gpu.WaitGroup
		for i, l := range textureHeader.MipLevels {
			wg.Add(1)

			var jq gpu.JobQueue

			tmp := gpu.MakeSliceUncached[byte](int(l.Len))
			enqueueReadAt(&jq, f.(io.ReaderAt), tmp, int64(l.Off))

			img := t.Image.SubImage(
				t.Image.Dim(),
				t.Image.Format(),
				0, layers,
				i, i+1)
			img.EnqueueInit(&jq)

			gpu.EnqueueCopyMemoryToImage(&jq,
				img, [3]int{},
				tmp, 0, 0,
				img.Extent())

			jq.Cleanup(img.Destroy)

			jq.Cleanup(func() { gpu.Free(gpu.UnsafePointer(gpu.SliceData(tmp))) })

			wg.EnqueueDone(&jq)
		}
		// Wait for upload to complete before closing the file. TODO: spawn a
		// goroutine to close the file instead, and just have the renderer block
		// on fence.
		wg.Wait()

		texturecache[filename] = t
	}
	return t
}

func model(geo game.GeometryPacked) *renderer.Mesh {
	m, ok := modelcache[geo]
	if !ok {
		unpacked := geo.Unpack()
		f, err := game.Data.Open(unpacked.Filename)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		m = new(renderer.Mesh)
		if err := m.InitFromFile(f.(io.ReaderAt), unpacked.Filename); err != nil {
			panic(err)
		}
		modelcache[geo] = m
	}
	return m
}

// TODO: move this into gpu?
func enqueueReadAt(jq *gpu.JobQueue, r io.ReaderAt, p gpu.Slice[byte], off int64) {
	enqueueHostCall(jq, func() {
		if _, err := r.ReadAt(p.Value(), off); err != nil {
			// We don't really have any way to report read failures for now.
			panic(err)
		}
	})
}

// TODO: move into gpu?
// TODO: come up with a better name?
func enqueueHostCall(jq *gpu.JobQueue, f func()) {
	var wg1 gpu.WaitGroup
	wg1.Add(1)
	var wg2 gpu.WaitGroup
	wg2.Add(1)

	wg1.EnqueueDone(jq)
	go func() {
		wg1.Wait()
		f()
		wg2.Done()
	}()
	wg2.EnqueueWait(jq)
}
