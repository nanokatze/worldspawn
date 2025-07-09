package effect

type Effect struct {
}

func (eff *Effect) ReadAt(p []byte, off int64) (int, error) {
	return 0, nil
}
