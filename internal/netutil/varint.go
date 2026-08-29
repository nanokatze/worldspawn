// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package netutil

import "io"

func consumeVarint(b []byte) (uint64, int) {
	if len(b) < 1 {
		return 0, -1
	}
	b0 := b[0] & 0x3f
	switch b[0] >> 6 {
	case 0:
		return uint64(b0), 1
	case 1:
		if len(b) < 2 {
			return 0, -1
		}
		return uint64(b0)<<8 | uint64(b[1]), 2
	case 2:
		if len(b) < 4 {
			return 0, -1
		}
		return uint64(b0)<<24 | uint64(b[1])<<16 | uint64(b[2])<<8 | uint64(b[3]), 4
	case 3:
		if len(b) < 8 {
			return 0, -1
		}
		return uint64(b0)<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 | uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7]), 8
	}
	return 0, -1
}

// appendVarint appends a variable-length integer to b.
//
// https://www.rfc-editor.org/rfc/rfc9000.html#section-16
func appendVarint(b []byte, v uint64) []byte {
	switch {
	case v <= 1<<6-1:
		return append(b, byte(v))
	case v <= 1<<14-1:
		return append(b, (1<<6)|byte(v>>8), byte(v))
	case v <= 1<<30-1:
		return append(b, (2<<6)|byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
	case v <= 1<<62-1:
		return append(b, (3<<6)|byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32), byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
	default:
		panic("varint too large")
	}
}

func readVarint(r io.Reader, tmp []byte) (uint64, error) {
	b := tmp[:8]
	if _, err := r.Read(b[:1]); err != nil {
		return 0, err
	}
	b = b[:1<<(b[0]>>6)]
	if _, err := r.Read(b[1:]); err != nil {
		return 0, err
	}
	v, _ := consumeVarint(b)
	return v, nil
}

func writeVarint(w io.Writer, v uint64, tmp []byte) error {
	_, err := w.Write(appendVarint(tmp[:0], v))
	return err
}
