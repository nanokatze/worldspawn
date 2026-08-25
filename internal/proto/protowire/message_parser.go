package protowire

import "errors"

type MessageParser []byte

func (p *MessageParser) ConsumeRecord() (Record, error) {
	recordHeader, n := consumeRecordHeader(*p)
	if n <= 0 {
		return Record{}, errors.New("failed to parse record header")
	}
	*p = (*p)[n:]

	payloadLen := recordHeader.PayloadLen

	if !(payloadLen <= len(*p)) {
		return Record{}, errors.New("truncated message")
	}

	payload := (*p)[:payloadLen]
	*p = (*p)[payloadLen:]
	return Record{recordHeader.Tag, payload}, nil
}
