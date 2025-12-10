package main

import (
	"io"
	"io/fs"
	"log"
	"path"
	"reflect"
	"sync"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
	"worldspawn/image/ktx2"
	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
	"worldspawn/internal/mtlj"
	"worldspawn/internal/renderer"
	"worldspawn/internal/renderer/matc"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// TODO: rename this file to something else
// TODO: outline this into its own package

var texturecache = make(map[string]*renderer.Texture)
var materialcache = make(map[string]renderer.MaterialInstance)
var modelcache = make(map[game.GeometryPacked]*mymesh)

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

func getmaterial(identifier string) renderer.MaterialInstance {
	m, ok := materialcache[identifier]
	if !ok {
		log.Println("loading material", path.Clean(identifier))

		src, err := fs.ReadFile(game.Data, path.Clean(identifier))
		if err != nil {
			log.Printf("getmaterial: %v", err)
			goto bail
		}

		// TODO: naming!!!!!!!!!!!!!!!!

		var header struct {
			ParamTypes []string
			Host       []string
			Program    jsontext.Value
		}
		if err := json.Unmarshal(src, &header); err != nil {
			log.Printf("getmaterial: %v", err)
			goto bail
		}

		paramTypes := make([]compiler.Type, len(header.ParamTypes))
		for i := range paramTypes {
			paramTypes[i] = mtlj.Type(header.ParamTypes[i])
		}
		paramStruct := matc.ParamStruct(paramTypes)
		paramOffsets := make([]int64, paramStruct.NumField())
		for i := range paramOffsets {
			paramOffsets[i] = int64(paramStruct.Field(i).Offset)
		}

		sea := compiler.NewSea()
		b := &compiler.Builder{
			Sea:   sea,
			Rules: append(append([]compiler.RewriteRule(nil), core.Rules...), matc.InterpreterLowerings...),
		}
		ir, err := mtlj.Parse(b, header.Program)
		if err != nil {
			log.Printf("getmaterial: %v", err)
			goto bail
		}

		m.Material = renderer.NewInterpretedMaterial(matc.CompileInterpretedMaterial(paramOffsets, sea, ir))
		m.Material.ParamStruct = paramStruct
		m.Material.ParamNames = header.Host
	}
	materialcache[identifier] = m
	return m

bail:
	// TODO: stop using gotos lmao aaa
	m.Material = errorMaterial()
	m.Material.ParamStruct = reflect.StructOf(nil)
	materialcache[identifier] = m
	return m
}

var errorMaterial = sync.OnceValue(func() *renderer.InterpretedMaterial {
	sea := compiler.NewSea()
	b := &compiler.Builder{
		Sea:   sea,
		Rules: append(append([]compiler.RewriteRule(nil), core.Rules...), matc.InterpreterLowerings...),
	}

	/*
		emissionSpectrum := core.MakeArray(b,
			core.Int32,
			core.IConst(b, core.Int32, int64(math.Float32bits(1))),
			core.IConst(b, core.Int32, int64(math.Float32bits(0))),
			core.IConst(b, core.Int32, int64(math.Float32bits(1))))
	*/
	program := b.Value2(
		matc.OpMakeSurface,
		core.EmptyType{},
		nil,
		// bsdf,
		b.Value2(matc.OpDFWeightedSum, matc.BSDFType{}, nil),
		// edf
		b.Value2(matc.OpDFWeightedSum, matc.EDFType{}, nil),
	)

	return renderer.NewInterpretedMaterial(matc.CompileInterpretedMaterial(nil, sea, program))
})

type mymesh struct {
	re *renderer.Mesh
}

func loadmesh(filename string) *mymesh {
	f, err := game.Data.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	tmp := new(renderer.Mesh)
	if err := tmp.InitFromFile(f.(io.ReaderAt), filename, getmaterial); err != nil {
		panic(err)
	}
	return &mymesh{tmp}
}

func getmesh(geo game.GeometryPacked) *mymesh {
	m, ok := modelcache[geo]
	if !ok {
		m = loadmesh(geo.Unpack().Filename)
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
