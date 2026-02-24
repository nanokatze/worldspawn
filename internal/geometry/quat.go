package geometry

import "golang.org/x/exp/constraints"

var quatMulTab = [4][4]uint8{
	/*       i     j     k     1 */
	/*i*/ {0x83, 0x02, 0x81, 0x00},
	/*j*/ {0x82, 0x83, 0x00, 0x01},
	/*k*/ {0x01, 0x80, 0x83, 0x02},
	/*1*/ {0x00, 0x01, 0x02, 0x03},
}

type quat[T constraints.Float] [4]T

func quatFromVec3[T constraints.Float](imag vec3[T]) quat[T] {
	return quat[T]{imag[0], imag[1], imag[2]}
}

func (q quat[T]) Conj() quat[T] {
	return quat[T]{-q[0], -q[1], -q[2], q[3]}
}

// TODO: rewrite this to be clearer and probably unroll with a generator.
func (p quat[T]) Mul(q quat[T]) (pq quat[T]) {
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			k := quatMulTab[i][j] & 3
			s := 1 - 2*T(quatMulTab[i][j]>>7)
			pq[k] += s * (p[i] * q[j])
		}
	}
	return pq
}

func (q quat[T]) Imag() vec3[T] {
	return vec3[T]{q[0], q[1], q[2]}
}
