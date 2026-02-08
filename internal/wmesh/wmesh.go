package wmesh

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/go-json-experiment/json"
)

type Attribute struct {
	Name   string
	Type   string // TODO: maybe replace with an enum?
	Domain int64  // TODO: replace with an enum
	Data   int64
}

// A range of primitives
type Range struct {
	MaterialIndex int64 // TODO: should be explicit from where this struct appears
	First         int64
	Count         int64
}

type Header struct {
	PrimitiveCount int64

	VertexCount int64

	IndexType   string // TODO: replace with an enum
	IndexBuffer int64

	// Attributes.
	//
	// Attributes must appear in order of non-decreasing Data.
	//
	// TODO: explicitly specify material index attribute? For Blender this would
	// be material_index.
	Attributes []Attribute

	Materials []string

	// Ad-hoc structures follow

	// Ranges of primitives with the same material indices.
	RangesByMaterialIndex []Range
}

type Section struct {
	Off, Size int64
}

type Preamble struct {
	Magic  [16]byte // "Worldspawn"
	Magic2 [16]byte // "Mesh"
	Header Section
	Blob   Section
}

type File struct {
	Preamble Preamble
	Header   Header
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

	var header Header
	if err := json.UnmarshalRead(io.NewSectionReader(r, preamble.Header.Off, preamble.Header.Size), &header, json.StringifyNumbers(true)); err != nil {
		return nil, err
	}

	return &File{preamble, header, r}, nil
}
