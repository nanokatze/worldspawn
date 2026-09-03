package animation

import (
	"errors"
	"strconv"
	"strings"
)

type rat128 struct {
	a int64
	b int64
}

func (r *rat128) UnmarshalText(s []byte) error {
	var err error

	a, b, ok := strings.Cut(string(s), "/")

	if ok {
		if r.a, err = strconv.ParseInt(a, 10, 64); err != nil {
			return err
		}
		if r.b, err = strconv.ParseInt(b, 10, 64); err != nil {
			return err
		}
		if !(r.b > 0) {
			return errors.New("invalid") // TODO: be more informative
		}
		// TODO: reduce r here
		return nil
	}

	r.a, err = strconv.ParseInt(a, 10, 64)
	r.b = 1
	return err
}
