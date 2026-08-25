package protowire

const MaxPayloadLen = 1<<31 - 1

type Value struct {
	Type    Type
	Payload []byte
}
