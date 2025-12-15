package wmesh

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/go-json-experiment/json"
)

type GeometryHeader struct {
	Materials []string
	// TODO: inline it into GeometryHeader directly and have the physics geometry be
	// "simplified"? (and eventually removed)
	Rendering RenderingGeometryHeader
	// Collision CollisionGeometryHeader
}

type RenderingGeometryHeader struct {
	Attributes []AttributeDesc
	Parts      []PartHeader
}

type AttributeDesc struct {
	Name string
	Type string
}

type PartHeader struct {
	MaterialIndex int
	AttribBuffers []int64
	VertexCount   int
	IndexType     string
	IndexBuffer   int64
	TriangleCount int
}

type Section struct {
	Off, Size int64
}

type Preamble struct {
	Magic  [16]byte // "Worldspawn"
	Magic2 [16]byte // "Geometry"
	Header Section
	Blob   Section
}

type File struct {
	Preamble Preamble
	Header   GeometryHeader
	R        io.ReaderAt
}

// TODO: I think a good idea would be to have "read header" and "open file"
// functions. "open file" would take io.ReaderAt and keep it around.

func NewFile(r io.ReaderAt) (*File, error) {
	// TODO: switch from preamble + json setup, to just using nice with schema
	// as prefix and nothing else.

	var preamble Preamble
	if err := binary.Read(io.NewSectionReader(r, 0, math.MaxInt64), binary.LittleEndian, &preamble); err != nil {
		return nil, err
	}

	var header GeometryHeader
	if err := json.UnmarshalRead(io.NewSectionReader(r, preamble.Header.Off, preamble.Header.Size), &header, json.StringifyNumbers(true)); err != nil {
		return nil, err
	}

	return &File{preamble, header, r}, nil
}
