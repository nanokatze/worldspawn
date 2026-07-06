package replication

import (
	"unique"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/internal/nice"
)

var NiceOptions = nice.WithArshalers(nice.JoinArshalers(
	uniqueHandleArshaler[string](),
	nice.InterfaceArshaler[game.Entity](game.EntityTypes...),
	nice.InterfaceArshaler[game.InputCmd](game.InputCmdTypes...)))

func uniqueHandleArshaler[T comparable]() nice.Arshalers {
	return nice.MakeArshaler(
		func(enc *nice.Encoder, v *unique.Handle[T]) error {
			tmp := v.Value()
			return nice.MarshalEncode(enc, &tmp)
		},
		func(dec *nice.Decoder, v *unique.Handle[T]) error {
			var tmp T
			if err := nice.UnmarshalDecode(dec, &tmp); err != nil {
				return err
			}
			*v = unique.Make(tmp)
			return nil
		},
	)
}
