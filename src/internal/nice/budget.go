package nice

import (
	"fmt"
	"reflect"
)

type outOfBudgetError struct {
	N, Budget int
}

func (e *outOfBudgetError) Error() string {
	return fmt.Sprintf("out of memory: %d (remaining budget %d)", e.N, e.Budget)
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
		return &outOfBudgetError{n, budget.n}
	}
	budget.n -= n
	return nil
}

func accountT(budget *Budget, t reflect.Type, n int) error {
	return budget.Account(n * int(t.Size()))
}
