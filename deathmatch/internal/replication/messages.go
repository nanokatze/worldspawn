package replication

// TODO: make things more typed than this. http2 frames would probably be kinda
// what we want.
const (
	_ = iota

	ResetTicker
	ResetWorld
	UpdateWorld
	SetPlayer
)

// methods called by client, on the server
const (
	_ = iota

	// C2SInput
)
