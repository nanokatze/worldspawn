package ecs

type ID uint64

const NullID = ID(0)

func MakeID(index int, generation uint32) ID {
	id := ID(uint64(index) | uint64(generation)<<32)
	if id == NullID {
		panic("null id")
	}
	// TODO: more detailed diagnostic
	if id.Index() != index || id.Generation() != generation {
		panic("index and/or generation out of range")
	}
	return id
}

func (id ID) Index() int {
	return int(uint64(id) & 0xffffffff)
}

// TODO: we could make this an int perhaps?
func (id ID) Generation() uint32 {
	return uint32(uint64(id) >> 32)
}

func (id ID) Succ() ID {
	return id + 1 // TODO: this is incorrect
}

// TODO: String method?
