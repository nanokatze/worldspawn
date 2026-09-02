package ktx2 // TODO: implement common handling in gpu/image like Go's core image.Decode does?

var magic = [12]byte{0xAB, 0x4B, 0x54, 0x58, 0x20, 0x32, 0x30, 0xBB, 0x0D, 0x0A, 0x1A, 0x0A}

type fileHeader struct {
	Magic                      [12]byte
	Format                     uint32
	TypeSize                   uint32
	Extent                     [3]uint32
	LayerCount                 uint32
	FaceCount                  uint32
	LevelCount                 uint32
	SupercompressionScheme     uint32
	DataFormatDescriptor       indexEntry32
	KeyValueData               indexEntry32
	SupercompressionGlobalData indexEntry64
}

type indexEntry32 struct {
	Offset uint32
	Length uint32
}

type indexEntry64 struct {
	Offset uint64
	Length uint64
}

type levelIndexEntry struct {
	indexEntry64
	UncompressedLength uint64
}
