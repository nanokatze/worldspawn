package main

import (
	"encoding/binary"
	"io"
	"io/fs"
	"log"
	"math"
	"path"
	"reflect"
	"sync"
	"unsafe"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
	"worldspawn/image/ktx2"
	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
	"worldspawn/internal/pathtracer"
	"worldspawn/internal/pathtracer/matc"
	"worldspawn/internal/wmat"
	"worldspawn/internal/wmesh"

	"github.com/go-json-experiment/json"
)

// TODO: rename this file to something else
// TODO: outline this into its own package

var texturecache = make(map[string]*pathtracer.Texture)
var materialcache = make(map[string]pathtracer.MaterialInstance)
var modelcache = make(map[game.GeometryPacked]*fileBackedMesh)

// TODO: should support streaming etc.
func texture(filename string) *pathtracer.Texture {
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

		t = pathtracer.NewTexture(
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

func getmaterial(identifier string) pathtracer.MaterialInstance {
	m, ok := materialcache[identifier]
	if !ok {
		log.Println("loading material", path.Clean(identifier))

		src, err := fs.ReadFile(game.Data, path.Clean(identifier))
		if err != nil {
			log.Printf("getmaterial: %v", err)
			goto bail
		}

		// TODO: naming!!!!!!!!!!!!!!!!

		var header wmat.Header
		if err := json.Unmarshal(src, &header); err != nil {
			log.Printf("getmaterial: %v", err)
			goto bail
		}

		paramTypes := make([]compiler.Type, len(header.ParamTypes))
		for i := range paramTypes {
			paramTypes[i] = wmat.Type(header.ParamTypes[i])
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
		ir, err := wmat.Parse(b, header.Program)
		if err != nil {
			log.Printf("getmaterial: %v", err)
			goto bail
		}

		// TODO: probs actually move GatherArgs into the renderer? idk
		m.Material = pathtracer.NewInterpretedMaterial(matc.CompileInterpretedMaterial(paramOffsets, sea, ir))
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

var errorMaterial = sync.OnceValue(func() *pathtracer.InterpretedMaterial {
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

	return pathtracer.NewInterpretedMaterial(matc.CompileInterpretedMaterial(nil, sea, program))
})

type fileBackedMesh struct {
	re *pathtracer.Mesh
}

func loadmesh(filename string) *fileBackedMesh {
	f, err := game.Data.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	r := f.(io.ReaderAt)

	var preamble wmesh.Preamble
	if err := binary.Read(io.NewSectionReader(r, 0, math.MaxInt64), binary.LittleEndian, &preamble); err != nil {
		panic(err)
	}

	var header2 wmesh.GeometryHeader
	if err := json.UnmarshalRead(io.NewSectionReader(r, preamble.Header.Off, preamble.Header.Size), &header2, json.StringifyNumbers(true)); err != nil {
		panic(err)
	}

	blob2 := io.NewSectionReader(r, preamble.Blob.Off, preamble.Blob.Size)

	ptmesh := new(pathtracer.Mesh)
	ptmesh.Parts = make([]pathtracer.MeshPart, len(header2.Rendering.Parts))
	// TODO: kill
	ptmesh.DefaultMaterials = make([]pathtracer.MaterialInstance, len(header2.Rendering.Parts))
	for i, serializedPart := range header2.Rendering.Parts {
		part := &ptmesh.Parts[i]

		part.PosBuffer = gpu.MakeSliceUncached[[3]float32](serializedPart.VertexCount)
		part.NormalBuffer = gpu.MakeSliceUncached[[3]float32](serializedPart.VertexCount)
		part.AttribBuffers = []any{
			gpu.MakeSliceUncached[[2]float32](serializedPart.VertexCount),
		}
		part.IndexBuffer = gpu.MakeSliceUncached[[3]uint16](serializedPart.TriangleCount)

		if _, err := blob2.ReadAt(byteslice(part.PosBuffer.Value()), serializedPart.PosBuffer); err != nil {
			panic(err)
		}
		if _, err := blob2.ReadAt(byteslice(part.NormalBuffer.Value()), serializedPart.NormalBuffer); err != nil {
			panic(err)
		}
		if _, err := blob2.ReadAt(byteslice(part.AttribBuffers[0].(gpu.Slice[[2]float32]).Value()), serializedPart.AttribBuffers[0]); err != nil {
			panic(err)
		}

		if _, err := blob2.ReadAt(byteslice(part.IndexBuffer.Value()), serializedPart.IndexBuffer); err != nil {
			panic(err)
		}

		ptmesh.DefaultMaterials[i] = getmaterial(header2.Materials[serializedPart.MaterialIndex])
	}

	ptmesh.InitAccel()

	var jq gpu.JobQueue
	ptmesh.BuildAccel(&jq)
	jq.WaitForIdle()

	return &fileBackedMesh{ptmesh}
}

func byteslice[T any](s []T) []byte {
	sizeofT := int(unsafe.Sizeof(*new(T)))
	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(s))), len(s)*sizeofT)
}

func getmesh(geo game.GeometryPacked) *fileBackedMesh {
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
