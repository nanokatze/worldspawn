package gputext

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"

	"golang.org/x/image/vector"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

type Font struct {
	mu     sync.Mutex
	glyphs map[font.GID]*Glyph // replace with sync.Map?

	scale float32
	face  *font.Face
}

// TODO: make Glyph internals private
type Glyph struct {
	OffX, OffY float32 // these are in *normalized* coords
	Data       *gpu.Image
}

// To match scale with the Size you specify in shaper, use float32(Size) /
// float32(face.Upem())
func NewFont(face *font.Face, scale float32) *Font {
	// TODO: make scale be relative to Upem? Would be nice for it to match with
	// the size we pass to the shaper.
	// TODO: to get all the glyphs we'd have to manually iterate over the Glyf
	// table, for which we'd use *opentype.Loader
	return &Font{
		glyphs: make(map[font.GID]*Glyph),

		scale: scale,
		face:  face,
	}
}

func (f *Font) Glyph(gid font.GID) *Glyph {
	f.mu.Lock()
	defer f.mu.Unlock()

	if glyph, ok := f.glyphs[gid]; ok {
		return glyph
	}

	data := f.face.GlyphData(gid)
	switch data := data.(type) {
	case font.GlyphOutline:
		if len(data.Segments) == 0 {
			return nil
		}

		// TODO: can we replace this with just getting the glyph extents?
		minX, minY, maxX, maxY := outlineBounds(data)

		minX *= f.scale
		minY *= f.scale
		maxX *= f.scale
		maxY *= f.scale

		// TODO: floor min* and ceil max* respectively here

		_ = minX + minY + maxX + maxY

		width := int(math.Ceil(float64(-minX + maxX)))
		height := int(math.Ceil(float64(-minY + maxY)))

		rast := vector.NewRasterizer(width, height)
		for _, seg := range data.Segments {
			switch seg.Op {
			case opentype.SegmentOpMoveTo:
				a := seg.Args[0]
				a.X = a.X*f.scale - minX
				a.Y = a.Y*f.scale - minY
				rast.MoveTo(a.X, a.Y)
			case opentype.SegmentOpLineTo:
				b := seg.Args[0]
				b.X = b.X*f.scale - minX
				b.Y = b.Y*f.scale - minY
				rast.LineTo(b.X, b.Y)
			case opentype.SegmentOpQuadTo:
				b := seg.Args[0]
				b.X = b.X*f.scale - minX
				b.Y = b.Y*f.scale - minY
				c := seg.Args[1]
				c.X = c.X*f.scale - minX
				c.Y = c.Y*f.scale - minY
				rast.QuadTo(b.X, b.Y, c.X, c.Y)
			case opentype.SegmentOpCubeTo:
				b := seg.Args[0]
				b.X = b.X*f.scale - minX
				b.Y = b.Y*f.scale - minY
				c := seg.Args[1]
				c.X = c.X*f.scale - minX
				c.Y = c.Y*f.scale - minY
				d := seg.Args[2]
				d.X = d.X*f.scale - minX
				d.Y = d.Y*f.scale - minY
				rast.CubeTo(b.X, b.Y, c.X, c.Y, d.X, d.Y)
			default:
				panic("unreachable")
			}
		}
		rast.ClosePath()

		gpuImg := gpu.NewImage(&gpu.ImageConfig{
			Dim:       gpu.ImageDim2D,
			Extent:    gpu.Int3{rast.Size().X, rast.Size().Y, 1},
			Layers:    1,
			MipLevels: 1,
			Samples:   1,
			Format:    vk.FORMAT_R8_UNORM,
		})

		var jq gpu.JobQueue

		pix := gpu.MakeSliceUncached[uint8](1 * rast.Size().X * rast.Size().Y)
		defer jq.Cleanup(func() { gpu.Free(gpu.UnsafePointer(gpu.SliceData(pix))) })

		rast.Draw(
			&image.Alpha{
				Pix:    pix.Value(),
				Stride: 1 * rast.Size().X,
				Rect:   rast.Bounds(),
			},
			rast.Bounds(),
			image.NewUniform(color.White),
			image.Point{})

		gpu.EnqueueCopyMemoryToImage(&jq,
			gpuImg, gpu.Int3{},
			pix, 0, 0,
			gpuImg.Extent())

		jq.WaitForIdle()

		glyph := &Glyph{
			OffX: minX / float32(rast.Size().X),
			OffY: minY / float32(rast.Size().Y),
			Data: gpuImg,
		}
		f.glyphs[gid] = glyph
		return glyph

	default:
		panic("unsupported glyph data type")
	}
}

func outlineBounds(outline font.GlyphOutline) (minX, minY, maxX, maxY float32) {
	minX = float32(math.Inf(1))
	minY = float32(math.Inf(1))
	maxX = float32(math.Inf(-1))
	maxY = float32(math.Inf(-1))
	for _, seg := range outline.Segments {
		for _, p := range seg.ArgsSlice() {
			minX = min(minX, p.X)
			minY = min(minY, p.Y)
			maxX = max(maxX, p.X)
			maxY = max(maxY, p.Y)
		}
	}
	return minX, minY, maxX, maxY
}
