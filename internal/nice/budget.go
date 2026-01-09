package nice

import "reflect"

type outOfBudgetError struct {
	n, budget int
}

func (e outOfBudgetError) Error() string { return "out of budget" }

type Budget struct {
	n int
}

func (budget *Budget) reset(n int) {
	budget.n = n
}

func (budget *Budget) Draw(n int) error {
	if n < 0 {
		panic("uhh?")
	}
	if budget.n < n {
		return outOfBudgetError{n, budget.n}
	}
	budget.n -= n
	return nil
}

func budgetDrawValues(budget *Budget, t reflect.Type, n int) error {
	return budget.Draw(n * int(t.Size()))
}

func budgetDrawMap(budget *Budget, t reflect.Type, n int) error {
	keyType := t.Key()
	valueType := t.Elem()
	return budget.Draw(n * int(keyType.Size()+valueType.Size()))
}
