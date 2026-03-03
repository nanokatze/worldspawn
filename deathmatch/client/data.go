package main

import (
	"encoding/binary"
	"io"
	"io/fs"
	"log"
	"math"
	"os"
	"path"
	"sync"
	"unsafe"

	"github.com/go-json-experiment/json"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/gpu"
	"worldspawn/gpu/imageio/ktx2"
	"worldspawn/gpu/vk"
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

		d, err := ktx2.NewDecoder(f.(io.ReaderAt))
		if err != nil {
			panic(err)
		}

		conf := d.Config()

		t = new(pathtracer.Texture)
		t.Image = gpu.NewImage(
			vk.Format(conf.Format),
			conf.Extent[:conf.Dim],
			gpu.WithLayers2(conf.Layers),
			gpu.WithMips2(conf.Mips),
			gpu.WithUsage(vk.IMAGE_USAGE_SAMPLED_BIT))
		if conf.Cube {
			t.Image = t.Image.SubImage(gpu.ViewAs(gpu.ImageDimCube))
		}

		var wg gpu.WaitGroup
		for i := range conf.Mips {
			wg.Add(1)

			jq := new(gpu.JobQueue)

			img := t.Image.SubImage(gpu.WithMips{i, i + 1})
			img.EnqueueInit(jq)

			d.EnqueueDecode(jq, img, i)

			jq.Cleanup(img.Destroy)

			wg.EnqueueDone(jq)
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
			Rules: append(append([]compiler.RewriteRule(nil), core.Rules...), matc.LowerToInterpreter...),
		}
		ir, err := wmaterial.Parse(b, header.Program)
		if err != nil {
			log.Printf("getmaterial: %v", err)
			goto bail
		}

		debuglog := io.Writer(nil)
		// This material bugs out
		if identifier == "weapons/grenade_launcher/materials/Anodized_Aluminium" {
			debuglog = os.Stderr
		}

		m.preamble = matc.CompilePreamble(paramsTuple, header.Preamble)
		m.material = pathtracer.NewInterpretedMaterial(matc.CompileInterpretedMaterial(paramsTuple, sea, ir, debuglog))
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
		Rules: append(append([]compiler.RewriteRule(nil), core.Rules...), matc.LowerToInterpreter...),
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
	joints []string

	// jointWeights gpu.Slice[struct {
	// 	Index  uint32
	// 	Weight float32
	// }] // we could get away just fine with unorm16, or even unorm8 in some cases.

	materials []string

	pathtracerMesh pathtracer.Mesh
}

// TODO: equip AttributeBuffer with size so we don't have to pull in vertex count
func loadattrbuf(blob2 io.ReaderAt, vertexCount int, desc wmesh.AttributeBuffer) any {
	if desc.Domain != wmesh.PerVertex {
		return nil
	}

	var ret any
	var bytes []byte
	switch desc.Type {
	case "R32G32B32_SFLOAT":
		guh := gpu.MakeSliceUncached[[3]float32](vertexCount)
		ret = guh
		bytes = byteslice(guh.Value())

	case "R32G32_SFLOAT":
		guh := gpu.MakeSliceUncached[[2]float32](vertexCount)
		ret = guh
		bytes = byteslice(guh.Value())

	default:
		panic("uhh")
	}

	if _, err := blob2.ReadAt(bytes, desc.Data.Data); err != nil {
		panic(err)
	}

	return ret
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

	var header2 wmesh.Header
	if err := json.UnmarshalRead(io.NewSectionReader(r, preamble.Header.Off, preamble.Header.Size), &header2, json.StringifyNumbers(true)); err != nil {
		panic(err)
	}

	blob2 := io.NewSectionReader(r, preamble.Blob.Off, preamble.Blob.Size)

	indexBuffer := gpu.MakeSliceUncached[[3]uint16](int(header2.PrimitiveCount))
	if _, err := blob2.ReadAt(byteslice(indexBuffer.Value()), header2.IndexBuffer.Data); err != nil {
		panic(err)
	}

	attrs := make([]any, 2)

	attrs[pathtracer.AttributePosition] = loadattrbuf(blob2, int(header2.VertexCount), header2.Positions)
	attrs[pathtracer.AttributeNormal] = loadattrbuf(blob2, int(header2.VertexCount), header2.Normals)

	// for i, desc := range header2.AttributeBuffers {
	// 	if desc.Domain != 0 {
	// 		continue
	// 	}

	// 	var bytes []byte
	// 	switch desc.Type {
	// 	case "R32G32B32_SFLOAT":
	// 		guh := gpu.MakeSliceUncached[[3]float32](int(header2.VertexCount))
	// 		bytes = byteslice(guh.Value())
	// 		attrs[i] = guh

	// 	case "R32G32_SFLOAT":
	// 		guh := gpu.MakeSliceUncached[[2]float32](int(header2.VertexCount))
	// 		bytes = byteslice(guh.Value())
	// 		attrs[i] = guh

	// 	default:
	// 		panic("uhh")
	// 	}

	// 	if _, err := blob2.ReadAt(bytes, desc.Data); err != nil {
	// 		panic(err)
	// 	}
	// }

	pathtracerMesh := pathtracer.Mesh{}
	pathtracerMesh.Parts = make([]pathtracer.MeshPart, len(header2.MaterialIndexRanges))

	materials := make([]string, len(header2.MaterialIndexRanges)) // TODO: eventually it would be nice if we could just directly use header2.Materials
	for materialIndex, range_ := range header2.MaterialIndexRanges {
		materials[materialIndex] = header2.Materials[range_.MaterialIndex]
		part := &pathtracerMesh.Parts[materialIndex]
		part.AttributeBuffers = attrs
		part.IndexBuffer = indexBuffer.Slice(int(range_.First), int(range_.First)+int(range_.Count))
	}

	pathtracerMesh.InitAccel()

	jq := new(gpu.JobQueue)
	pathtracerMesh.BuildAccel(jq)
	gpu.WaitForIdle(jq)

	return &fileBackedMesh{
		materials:      materials,
		pathtracerMesh: pathtracerMesh,
	}
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
