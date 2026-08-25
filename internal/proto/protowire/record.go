package protowire

type Record struct {
	Tag
	Payload []byte
}

func (record Record) header() recordHeader {
	return recordHeader{
		Tag:        record.Tag,
		PayloadLen: len(record.Payload),
	}
}

func (record Record) Value() Value {
	return Value{
		Type:    record.Type(),
		Payload: record.Payload,
	}
}
