package nice

import (
	"fmt"
	"reflect"
)

type Budget struct {
	remaining int
}

func (budget *Budget) reset(n int) {
	budget.remaining = n
}

func (budget *Budget) Account(n int) error {
	if n < 0 {
		panic("uhh?")
	}
	if budget.remaining < n {
		return fmt.Errorf("out of memory: %d (remaining budget %d)", n, budget.remaining)
	}
	budget.remaining -= n
	return nil
}

func accountT(budget *Budget, t reflect.Type, n int) error {
	return budget.Account(n * int(t.Size()))
}
