package gmath

import "golang.org/x/exp/constraints"

const i = 'i'
const j = 'j'
const k = 'k'

var quatMulTab = [4][4]int8{
	/*      1   i   j   k */
	/*1*/ {+1, +i, +j, +k},
	/*i*/ {+i, -1, +k, -j},
	/*j*/ {+j, -k, -1, +i},
	/*k*/ {+k, +j, -i, -1},
}

type quat[T constraints.Float] [4]T

func quatFromVec3[T constraints.Float](imag Vec3[T]) quat[T] {
	return quat[T]{0, imag[0], imag[1], imag[2]}
}

func (q quat[T]) Conj() quat[T] {
	return quat[T]{q[0], -q[1], -q[2], -q[3]}
}

// TODO: rewrite this to be clearer and probably unroll with a generator.
func (p quat[T]) Mul(q quat[T]) (pq quat[T]) {
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			x := quatMulTab[i][j]

			k := -1
			switch max(x, -x) {
			case 1:
				k = 0
			case 'i':
				k = 1
			case 'j':
				k = 2
			case 'k':
				k = 3
			}

			s := T(1)
			if x < 0 {
				s = -1
			}

			pq[k] += s * (p[i] * q[j])
		}
	}
	return pq
}

func (q quat[T]) Imag() Vec3[T] {
	return Vec3[T](q[1:4])
}
