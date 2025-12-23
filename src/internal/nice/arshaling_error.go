package nice

type ArshalingError struct {
	Index string
	Err   error
}

func (e ArshalingError) Error() string {
	return e.Index + ": " + e.Err.Error()
}

func (e ArshalingError) Unwrap() error {
	return e.Err
}
