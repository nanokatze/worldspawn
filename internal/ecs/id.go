package ecs

type ID uint64

func MakeID(index int, generation uint32) ID {
	id := ID(uint64(index) | uint64(generation)<<32)
	if id == 0 {
		panic("wtf")
	}
	if id.Index() != index || id.Generation() != generation {
		panic("wtf2")
	}
	return id
}

func (id ID) Index() int {
	return int(uint64(id) & 0x7fffffff)
}

// TODO: make it int32?
func (id ID) Generation() uint32 {
	return uint32(uint64(id) >> 32)
}
