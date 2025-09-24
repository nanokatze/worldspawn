package opusfile

type opusTags struct {
	userComments []string
	vendor       string
}

func parseTags(src []byte) (*opusTags, error) {
	if len(src) < 8 {
		return nil, ErrBadHeader
	}

	if string(src[0:8]) != "OpusTags" {
		return nil, ErrBadHeader
	}

	return &opusTags{}, nil
}
