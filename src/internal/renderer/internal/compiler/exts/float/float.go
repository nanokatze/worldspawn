package float

import "fmt"

type FloatType struct {
	E int
	M int
}

func (t FloatType) String() string { return fmt.Sprintf("Float[%d,%d]", t.E, t.M) }
