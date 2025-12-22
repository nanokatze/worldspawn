package nice

import "reflect"

type outOfBudgetError struct {
	n, budget int
}

func (e *outOfBudgetError) Error() string { return "out of budget" }

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

func accountForValues(budget *Budget, t reflect.Type, n int) error {
	return budget.Account(n * int(t.Size()))
}

func accountForMap(budget *Budget, t reflect.Type, n int) error {
	keyType := t.Key()
	valueType := t.Elem()
	return budget.Account(n * int(keyType.Size()+valueType.Size()))
}
