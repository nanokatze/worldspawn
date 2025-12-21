package ecs

type Entity uint64

func MakeEntity(index int, generation uint32) Entity {
	id := Entity(uint64(index) | uint64(generation)<<32)
	if id == 0 {
		panic("wtf")
	}
	if id.Index() != index || id.Generation() != generation {
		panic("wtf2")
	}
	return id
}

func (id Entity) Index() int {
	return int(uint64(id) & 0xffffffff)
}

func (id Entity) Generation() uint32 {
	return uint32(uint64(id) >> 32)
}
