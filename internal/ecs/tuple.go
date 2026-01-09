package ecs

type Tuple[T0, T1 any] struct {
	V1 T0
	V2 T1
}

type Tuple3[T0, T1, T2 any] struct {
	V1 T0
	V2 T1
	V3 T2
}
