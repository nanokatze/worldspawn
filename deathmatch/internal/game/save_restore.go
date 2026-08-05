package game

import (
	"encoding/json/v2"
	"io"
)

// TODO: we'd benefit from an additional step before Restore so we can know the
// min capacity necessary for this save.
func (world *World) Restore(r io.Reader) error {
	// TODO: we should deserialize into an intermediate structure and do various
	// checks first. I think ideally we'd not return an error if we ended up
	// modifying the Scene?
	//
	// TODO: we should zero out Scene before restoring I guess.

	if err := json.UnmarshalRead(r, world, JSONOptions); err != nil {
		return err
	}
	return nil
}

func (world *World) Save(w io.Writer) error {
	panic("not implemented")
}
