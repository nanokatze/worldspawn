package work

const HashSize = 32

// TODO: Exec should accept a larger context object instead of outDir.

// TODO: make it generic over what gets passed to Exec
type Action[T any] struct {
	Hash [HashSize]byte
	Exec func(T) error
}

/*
type Executor struct {
	lookupcache func(hash [HashSize]byte) bool
}

func (exec *Executor) Add(a *Action) error {
	return nil
}
*/
