package gputext

import (
	"os"
	"testing"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/shaping"
)

const pt = 14

func loadFont(name string) (*font.Face, *Font, error) {
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

	return face, NewFont(face, float32(pt)/float32(face.Upem())), nil
}

func TestXxx(t *testing.T) {
	face, facegpu, err := loadFont("../../fonts/Huglove.ttf")
	if err != nil {
		t.Fatal(err)
	}

	// var segmenter shaping.Segmenter

	var shaper shaping.HarfbuzzShaper
	shaper.SetFontCacheSize(4)

	// pixels per inch
	// ppi := 96 // TODO: multiply this by the window's display scale

	// var segmenter shaping.Segmenter
	// segmenter.Split()

	shapingOutput := shaper.Shape(shaping.Input{
		Text:      []rune("Test"),
		RunStart:  0,
		RunEnd:    99999,
		Direction: di.DirectionLTR,
		Face:      face,
		Size:      14,
	})

	//shapingOutput.Size

	for _, shapedGlyph := range shapingOutput.Glyphs {
		t.Logf("%#v", shapingOutput.ToFontUnit(shapedGlyph.XAdvance)*pt/float32(face.Upem()))
		if false {
			_ = facegpu.Glyph(shapedGlyph.GlyphID)
		}
	}
}
