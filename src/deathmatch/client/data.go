package main

import (
	"encoding/binary"
	"io"
	"io/fs"
	"log"
	"maps"
	"math"
	"path"
	"sync"
	"unsafe"

	"github.com/go-json-experiment/json"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
	"worldspawn/image/ktx2"
	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
	sfx "worldspawn/internal/fuckwwise"
	"worldspawn/internal/pathtracer"
	"worldspawn/internal/pathtracer/matc"
	"worldspawn/internal/wmaterial"
	"worldspawn/internal/wmesh"
)

// TODO: rename this file to something else
// TODO: outline this into its own package. pathtracerio?

var texturecache = make(map[string]*pathtracer.Texture)
var materialcache = make(map[string]material)
var modelcache = make(map[string]*fileBackedMesh)

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

type material struct {
	preamble matc.Preamble
	material *pathtracer.InterpretedMaterial
}

func getmaterial(identifier string) material {
	m, ok := materialcache[identifier]
	if !ok {
		log.Println("loading material", path.Clean(identifier))

		src, err := fs.ReadFile(game.Data, path.Clean(identifier))
		if err != nil {
			log.Printf("getmaterial: %v", err)
			goto bail
		}

		// TODO: naming!!!!!!!!!!!!!!!!

		var header wmaterial.Header
		if err := json.Unmarshal(src, &header); err != nil {
			log.Printf("getmaterial: %v", err)
			goto bail
		}

		params := make([]compiler.Type, len(header.Params))
		for i := range params {
			params[i] = wmaterial.Type(header.Params[i])
		}

		paramsTuple := matc.MakeParamsTuple(params)

		sea := compiler.NewSea()
		b := &compiler.Builder{
			Sea:   sea,
			Rules: append(append([]compiler.RewriteRule(nil), core.Rules...), matc.InterpreterLowerings...),
		}
		ir, err := wmaterial.Parse(b, header.Program)
		if err != nil {
			log.Printf("getmaterial: %v", err)
			goto bail
		}

		m.preamble = matc.CompilePreamble(paramsTuple, header.Preamble)
		m.material = pathtracer.NewInterpretedMaterial(matc.CompileInterpretedMaterial(paramsTuple, sea, ir, nil))
	}
	materialcache[identifier] = m
	return m

bail:
	// TODO: stop using gotos lmao aaa
	m.preamble = func(dst []byte, props matc.PropertyBag) {}
	m.material = errorMaterial()
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
		core.NothingType{},
		nil,
		// bsdf,
		b.Value2(matc.OpDFWeightedSum, matc.BSDFType{}, nil),
		// edf
		b.Value2(matc.OpDFWeightedSum, matc.EDFType{}, nil),
	)

	return pathtracer.NewInterpretedMaterial(matc.CompileInterpretedMaterial(matc.ParamsTuple{}, sea, program, nil))
})

type fileBackedMesh struct {
	materials []string

	// we could (should) just embed it tbh
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

	attributes := maps.Collect(
		func(yield func(string, int) bool) {
			for i, attr := range header2.Rendering.Attributes {
				yield(attr.Name, i)
			}
		})

	inner := new(pathtracer.Mesh)

	inner.PosBuffer = attributes["position"]
	inner.NormalBuffer = attributes["normal"]

	inner.Parts = make([]pathtracer.MeshPart, len(header2.Rendering.Parts))
	materials := make([]string, len(header2.Rendering.Parts))
	for i, partHeader := range header2.Rendering.Parts {
		part := &inner.Parts[i]

		part.AttribBuffers = make([]any, len(partHeader.AttribBuffers))
		for j, attr := range header2.Rendering.Attributes {
			var bytes []byte
			switch attr.Type {
			case "R32G32B32_SFLOAT":
				guh := gpu.MakeSliceUncached[[3]float32](partHeader.VertexCount)
				bytes = byteslice(guh.Value())
				part.AttribBuffers[j] = guh
			case "R32G32_SFLOAT":
				guh := gpu.MakeSliceUncached[[2]float32](partHeader.VertexCount)
				bytes = byteslice(guh.Value())
				part.AttribBuffers[j] = guh
			default:
				panic("uhh")
			}

			if _, err := blob2.ReadAt(bytes, partHeader.AttribBuffers[j]); err != nil {
				panic(err)
			}
		}

		part.IndexBuffer = gpu.MakeSliceUncached[[3]uint16](partHeader.TriangleCount)
		if _, err := blob2.ReadAt(byteslice(part.IndexBuffer.Value()), partHeader.IndexBuffer); err != nil {
			panic(err)
		}

		materials[i] = header2.Materials[partHeader.MaterialIndex]
	}

	inner.InitAccel()

	var jq gpu.JobQueue
	inner.BuildAccel(&jq)
	jq.WaitForIdle()

	return &fileBackedMesh{materials, inner}
}

func getmesh(geo string) *fileBackedMesh {
	m, ok := modelcache[geo]
	if !ok {
		m = loadmesh(geo)
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

func byteslice[T any](s []T) []byte {
	sizeofT := int(unsafe.Sizeof(*new(T)))
	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(s))), len(s)*sizeofT)
}

func readSamples(r io.Reader, format sfx.Format) ([]float32, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	switch format {
	case sfx.FORMAT_S16:
		bufSNORM16 := unsafe.Slice((*int16)(unsafe.Pointer(unsafe.SliceData(buf))), len(buf)/2)
		bufFLOAT32 := make([]float32, len(bufSNORM16))
		for i := range bufFLOAT32 {
			bufFLOAT32[i] = max(float32(bufSNORM16[i])/32767.0, -1)
		}
		return bufFLOAT32, nil

	case sfx.FORMAT_F32:
		bufFLOAT32 := unsafe.Slice((*float32)(unsafe.Pointer(unsafe.SliceData(buf))), len(buf)/4)
		return bufFLOAT32, nil

	default:
		panic("unsupported format")
	}
}

func extractChannel(s []float32, channels, channel int) []float32 {
	if channels == 1 {
		return s
	}

	s2 := make([]float32, len(s)/channels)
	for i := range s2 {
		s2[i] = s[i*channels+channel]
	}
	return s2
}
