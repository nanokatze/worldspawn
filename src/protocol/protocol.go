package protocol

// TODO: could we perhaps replace these with an interface {}?

// methods called by server, on client
const (
	_ = iota

	SetDeltaTime = 5
	SetPlayer    = 7
	ResetWorld   = 6
	UpdateWorld  = 9
)

// methods called by client, on the server
const (
	_ = iota

	C2SInput
)
