package main

import (
	"encoding/binary"
	"encoding/json/v2"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path"
	"slices"
	"sync"
	"unique"
	"unsafe"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/gpu"
	"worldspawn/gpu/image/ktx2"
	"worldspawn/gpu/vk"
	"worldspawn/internal/cache"
	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
	"worldspawn/internal/geometry"
	"worldspawn/internal/loaders/audio"
	"worldspawn/internal/loaders/audio/wav"
	"worldspawn/internal/loaders/material"
	"worldspawn/internal/loaders/wmesh"
	"worldspawn/internal/renderer"
	"worldspawn/internal/renderer/matc"
)

// TODO: rename this file to something else
// TODO: outline this into its own package. pathtracerio?
// TODO: support streaming at finer granularity

var texturecache = cache.New(func(filename unique.Handle[string]) *renderer.Texture {
	// TODO: move this code into its own func + handle errors and everything.

	f, err := game.Data.Open(filename.Value())
	if err != nil {
		panic(err)
	}
	defer f.Close()

	t := new(renderer.Texture)
	t.Image, err = ktx2.Decode(f.(io.ReaderAt), vk.ImageUsageFlags(vk.IMAGE_USAGE_SAMPLED_BIT))
	if err != nil {
		panic(err)
	}

	return t
})

type rendererMaterial struct {
	preamble matc.Preamble
	material *renderer.InterpretedMaterial
}

var materialcache = cache.New(func(filename unique.Handle[string]) rendererMaterial {
	log.Println("loading material", path.Clean(filename.Value()))

	intermediate, err := material.Load(game.Data, path.Clean(filename.Value()))
	if err != nil {
		log.Printf("getmaterial: %v", err)

		return rendererMaterial{
			material: errorMaterial(),
		}
	}

	debuglog := io.Writer(nil)
	// This material bugs out
	if filename == unique.Make("weapons/grenade_launcher/materials/Anodized_Aluminium") {
		debuglog = os.Stderr
	}

	paramsTuple := matc.MakeParamsTuple(slices.Collect(func(yield func(compiler.Type) bool) {
		for _, typ := range intermediate.Params {
			yield(material.Type(typ))
		}
	}))

	return rendererMaterial{
		preamble: matc.CompilePreamble(paramsTuple, intermediate.Preamble),
		material: renderer.NewInterpretedMaterial(matc.CompileInterpretedMaterial(paramsTuple, nil, intermediate.IR, debuglog)),
	}
})

var errorMaterial = sync.OnceValue(func() *renderer.InterpretedMaterial {
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

	return renderer.NewInterpretedMaterial(matc.CompileInterpretedMaterial(matc.ParamsTuple{}, sea, program, nil))
})

type fileBackedMesh struct {
	// TODO: outline this stuff into some kind of mesh structure that would be
	// usable by geometry processing code. I guess we could also shove geometry
	// processing code into pathtracer, but it feels like they should be
	// separate. Notably, geometry processing needs fancier representation than
	// pathtracer, and might have weird intermediate representations that the
	// pathtracer can't work with directly like e.g. triangles not being neatly
	// grouped by material index.

	attrs []any

	joints []unique.Handle[string]

	jointsPerVertex int

	jointWeights gpu.Slice[geometry.Uhh]

	materials []unique.Handle[string]

	geometry renderer.Geometry
	accel    gpu.BLAS
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

var modelcache = cache.New(func(filename unique.Handle[string]) *fileBackedMesh {
	f, err := game.Data.Open(filename.Value())
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

	indexBufferData := gpu.MakeSliceUncached[uint32](3 * int(header2.PrimitiveCount))
	if _, err := blob2.ReadAt(byteslice(indexBufferData.Value()), header2.IndexBuffer.Data); err != nil {
		panic(err)
	}
	indexBuffer := renderer.IndexBufferFromUint32Slice(indexBufferData)

	attrs := make([]any, 2)

	attrs[renderer.AttributePosition] = loadattrbuf(blob2, int(header2.VertexCount), header2.Positions)
	attrs[renderer.AttributeNormal] = loadattrbuf(blob2, int(header2.VertexCount), header2.Normals)

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

	var jointWeights gpu.Slice[geometry.Uhh]
	if n := int(header2.VertexCount) * int(header2.MaxInfluencesPerVertex); n > 0 {
		jointWeights = gpu.MakeSliceUncached[geometry.Uhh](n)
		if _, err := blob2.ReadAt(byteslice(jointWeights.Value()), header2.JointWeights.Data); err != nil {
			panic(err)
		}
	}

	geometry := renderer.Geometry{}
	geometry.AttributeBuffers = attrs
	geometry.Parts = make([]renderer.GeometryPart, len(header2.MaterialIndexRanges))

	materials := make([]string, len(header2.MaterialIndexRanges)) // TODO: eventually it would be nice if we could just directly use header2.Materials
	for materialIndex, range_ := range header2.MaterialIndexRanges {
		materials[materialIndex] = header2.Materials[range_.MaterialIndex]
		part := &geometry.Parts[materialIndex]
		part.IndexBuffer = indexBuffer.Slice(3*int(range_.First), 3*(int(range_.First)+int(range_.Count)))
	}

	accelConfig := geometry.AccelConfig()
	accel := gpu.NewBLAS(accelConfig.CalcSizes().Accel)

	jq := new(gpu.JobQueue)
	accel.EnqueueBuild(jq, accelConfig)
	gpu.WaitForIdle(jq)

	return &fileBackedMesh{
		materials: slices.Collect(func(yield func(unique.Handle[string]) bool) {
			for _, j := range header2.Materials {
				yield(unique.Make(j))
			}
		}),
		attrs: attrs,
		joints: slices.Collect(func(yield func(unique.Handle[string]) bool) {
			for _, j := range header2.Joints {
				yield(unique.Make(j))
			}
		}),
		jointsPerVertex: int(header2.MaxInfluencesPerVertex),
		jointWeights:    jointWeights,
		geometry:        geometry,
		accel:           accel,
	}
})

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

func readSamples(r io.Reader, format wav.Format) ([]float32, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	switch format {
	case wav.FORMAT_S16:
		bufSNORM16 := unsafe.Slice((*int16)(unsafe.Pointer(unsafe.SliceData(buf))), len(buf)/2)
		bufFLOAT32 := make([]float32, len(bufSNORM16))
		for i := range bufFLOAT32 {
			bufFLOAT32[i] = max(float32(bufSNORM16[i])/32767.0, -1)
		}
		return bufFLOAT32, nil

	case wav.FORMAT_F32:
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

var soundcache = cache.New(func(filename unique.Handle[string]) []float32 {
	f, err := game.Data.Open(filename.Value())
	if err != nil {
		// TODO: should be non-fatal
		panic(fmt.Sprintf("failed to open file %v", filename.Value()))
	}
	defer f.Close()

	reader, err := audio.NewReader(f.(io.ReaderAt))
	if err != nil {
		panic(err)
	}

	samples, _ := readSamples(reader, wav.Format(reader.Config().Format))
	effect := extractChannel(samples, reader.Config().Channels, 0)

	return effect
})
