package ktx2 // TODO: implement common handling in gpu/image like Go's core image.Decode does?

var magic = [12]byte{0xAB, 0x4B, 0x54, 0x58, 0x20, 0x32, 0x30, 0xBB, 0x0D, 0x0A, 0x1A, 0x0A}

type fileHeader struct {
	Magic                      [12]byte
	Format                     uint32
	TypeSize                   uint32
	Extent                     [3]uint32
	LayerCount                 uint32
	FaceCount                  uint32
	MipCount                   uint32
	SupercompressionScheme     uint32
	DataFormatDescriptor       sectionHeader32
	KeyValueData               sectionHeader32
	SupercompressionGlobalData sectionHeader
}

type sectionHeader32 struct {
	Offset uint32
	Length uint32
}

type sectionHeader struct {
	Offset uint64
	Length uint64
}

type mipSectionHeader struct {
	sectionHeader
	UncompressedLength uint64
}
