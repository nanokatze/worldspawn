package nice

import (
	"fmt"
	"reflect"
)

type outOfMemoryError struct {
	n      int
	budget int
}

func (e *outOfMemoryError) Error() string {
	return fmt.Sprintf("out of memory: %d (remaining budget %d)", e.n, e.budget)
}

type Budget struct {
	n int
}

func (budget *Budget) reset(n int) {
	budget.n = n
}

func (budget *Budget) Account(n int) error {
	if n < 0 {
		panic("uhh?")
	}
	if budget.n < n {
		return &outOfMemoryError{n, budget.n}
	}
	budget.n -= n
	return nil
}

func accountT(budget *Budget, t reflect.Type, n int) error {
	return budget.Account(n * int(t.Size()))
}
