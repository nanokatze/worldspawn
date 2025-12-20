package nice2

import "fmt"

// TODO: rename
type ArshalingError struct {
	Chain string
	Err   error
}

func (e ArshalingError) Error() string {
	// TODO: strings.Builder
	chain := e.Chain
	for {
		e, ok := e.Err.(ArshalingError)
		if !ok {
			break
		}
		chain += e.Chain
	}
	return fmt.Sprintf("%s: %v", e.Chain, e.Err)
}
