package wmesh

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/go-json-experiment/json"
)

type IndexType int8

const (
	IndexNone IndexType = iota
	IndexUint8
	IndexUint16
	IndexUint32
)

type Domain int8

const (
	PerVertex Domain = 0
	PerEdge   Domain = 1
	PerFace   Domain = 2
)

type Preamble struct {
	Magic  [32]byte // "Worldspawn"
	Header Section
	Blob   Section
}

type Section struct {
	Off, Size int64
}

type Header struct {
	PrimitiveCount int64

	VertexCount int64

	IndexType   int64
	IndexBuffer Buffer

	Positions AttributeBuffer
	Normals   AttributeBuffer

	Joints []string

	MaxInfluencesPerVertex int64

	// VertexCount * MaxInfluencesPerVertex of index uint32 × weight float32 pairs
	JointWeights Buffer

	Materials []string

	// MaterialIndices AttributeBuffer

	// Ranges of primitives with the same material indices.
	//
	// TODO: rename pls
	MaterialIndexRanges []Range

	NamedAttributes map[string]AttributeBuffer
}

type Buffer struct {
	Data int64
	Size int64
}

type AttributeBuffer struct {
	Domain Domain // TODO: should be represented with a string in json probably
	Type   string // TODO: maybe replace with an enum?
	Data   Buffer
}

// A range of primitives
// TODO: replace uses of this with [2]int probably
type Range struct {
	MaterialIndex int64 // TODO: should be explicit from where this struct appears
	First         int64
	Count         int64
}

type File struct {
	r      io.ReaderAt
	header Header
	blob   Section
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

	return &File{r, header, preamble.Blob}, nil
}
