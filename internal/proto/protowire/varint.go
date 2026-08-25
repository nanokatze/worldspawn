package protowire

import "math/bits"

const MaxVarintLen = 10

func AppendVarint(b []byte, v uint64) []byte {
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

func AppendPaddedVarints(b []byte, pad int, vs ...uint64) []byte {
	for len(vs) > 1 {
		v := vs[0]
		overhead := len(vs)*MaxVarintLen - pad
		n := max(VarintLen(v), MaxVarintLen-overhead)
		b = appendPaddedVarint(b, n, v)
		pad -= n
		vs = vs[1:]
	}
	return appendPaddedVarint(b, pad, vs[0])
}

func appendPaddedVarint(b []byte, pad int, v uint64) []byte {
	for range pad - 1 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

func ConsumeVarint(b []byte) (uint64, int) {
	v := uint64(0)
	n := 0
	for n = range min(len(b), MaxVarintLen) {
		more := b[n] >= 0x80
		v += uint64(b[n]&0x7f) << (7 * n)
		n++
		if !more {
			return v, n
		}
	}
	return 0, -n
}

func VarintLen(v uint64) int {
	return max((bits.Len64(v)+7-1)/7, 1)
}
