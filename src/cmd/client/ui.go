package main

import (
	"sync"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

var testVertMain = sync.OnceValue(func() *gpu.Func {
	return gpu.NewFunc(mustReadFile("shaders/test.spv"), vk.SHADER_STAGE_VERTEX_BIT, "vertMain")
})

var testFragMain = sync.OnceValue(func() *gpu.Func {
	return gpu.NewFunc(mustReadFile("shaders/test.spv"), vk.SHADER_STAGE_FRAGMENT_BIT, "fragMain")
})

/*
type UIDrawer struct {
	dot geometry.Vec2 // TODO: should be integer

	shaper shaping.Shaper
	scale  float32

	rp *gpu.RenderPass

	sampler gpu.Sampler
}

func drawLabel(d *UIDrawer, typeface string, pt float32, s string) {
	rp := d.rp

	face := getfont(typeface)

	runes := []rune(s)
	shapingOutput := d.shaper.Shape(shaping.Input{
		Text:      runes,
		RunStart:  0,
		RunEnd:    len(runes),
		Direction: di.DirectionLTR,
		Face:      face.face,
		Size:      fixed.Int26_6(face.face.Upem()), // TODO: specify the size we actually want here
	})

	// shapingOutput.ToFontUnit()

	unnormalizedToNormalized := geometry.Vec2{
		d.scale / float32(currentExtent.X),
		d.scale / float32(currentExtent.Y),
	}

	for _, shapedGlyph := range shapingOutput.Glyphs {
		faceScale := pt / float32(face.face.Upem())

		// TODO: use ToFontUnit instead of doing math with faceScale by
		// ourselves.

		graphics := face.gpu.Glyph(shapedGlyph.GlyphID)
		if graphics != nil {
			tmp := graphics.Data.NewSamplingView()

			offsetN := geometry.Vec2{graphics.OffX, graphics.OffY}
			extent := geometry.Vec2{float32(shapedGlyph.Width), float32(shapedGlyph.Height)}.
				Scale(faceScale)

			rp.SetShaderArgs(&struct {
				Offset geometry.Vec2
				Extent geometry.Vec2
				Desc   gpu.SamplingViewWithSampler
			}{
				Offset: d.dot.Add(offsetN.Mul(extent)).Mul(unnormalizedToNormalized),
				Extent: extent.Mul(unnormalizedToNormalized),
				Desc:   tmp.WithSampler(d.sampler),
			})
			rp.Draw(6, 1, 0, 0)

			rp.Cleanup(tmp.Destroy)
		}

		d.dot = d.dot.Add(geometry.Vec2{float32(shapedGlyph.XAdvance), float32(shapedGlyph.YAdvance)}.Scale(faceScale))
	}
}

type fontcacheentry struct {
	face *font.Face
	gpu  *gputext.Font
}

var fontcache = make(map[string]fontcacheentry)

func getfont(name string) fontcacheentry {
	entry, ok := fontcache[name]
	if !ok {
		face, gpu, err := loadFont(name, 2*14)
		if err != nil {
			panic(err)
		}
		entry = fontcacheentry{face, gpu}
		fontcache[name] = entry
	}
	return entry
}

// TODO: do something about this
func loadFont(name string, pt float32) (*font.Face, *gputext.Font, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	ol, err := opentype.NewLoader(f)
	if err != nil {
		return nil, nil, err
	}

	ft, err := font.NewFont(ol)
	if err != nil {
		return nil, nil, err
	}

	// At this point we can close f.

	face := font.NewFace(ft)

	return face, gputext.NewFont(face, pt/float32(face.Upem())), nil
}
*/
