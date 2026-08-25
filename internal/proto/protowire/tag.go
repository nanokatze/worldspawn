package protowire

const LastFieldNumber = 1<<29 - 1

type Tag uint32

func PackTag(fieldNumber int, typ Type) Tag {
	if !(0 < fieldNumber && fieldNumber <= LastFieldNumber) {
		panic("bad")
	}

	var tagBits uint32
	tagBits |= uint32(fieldNumber) << 3
	tagBits |= uint32(typ)
	return Tag(tagBits)
}

func (t Tag) Type() Type { return Type(t & ((1 << 3) - 1)) }

func (t Tag) FieldNumber() int { return int(t >> 3) }
