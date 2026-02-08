package wmesh

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/go-json-experiment/json"
)

type Attribute struct {
	Name   string
	Type   string
	Domain int64
	Data   int64
}

type Part struct {
	MaterialIndex  int64 // TODO: should be explicit from where this Part appears
	FirstPrimitive int64
	PrimitiveCount int64
}

type GeometryHeader struct {
	PrimitiveCount int64

	VertexCount int64

	IndexType   string
	IndexBuffer int64

	Attributes []Attribute // TODO: explicitly specify material index attribute? For Blender this would be material_index.

	Materials []string // TODO: move towards the end but before ad-hoc structures?

	// Ad-hoc structures follow

	// TODO: rename
	PartitionByMaterialIndex []Part
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
