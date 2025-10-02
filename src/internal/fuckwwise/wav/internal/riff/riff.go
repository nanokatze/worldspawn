package riff

type Chunk struct {
	Id     [4]byte
	Size   uint32
	Format [4]byte
}

type Subchunk struct {
	Id   [4]byte
	Size uint32
}
