package replication

import (
	"crypto/sha256"
	"fmt"
	"io"
	"reflect"
	"unique"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/internal/nice"
)

func NiceOptions2(world *game.World) nice.Options {
	return nice.WithArshalers(nice.JoinArshalers(
		nice.MakeArshaler[unique.Handle[string]](
			func(enc *nice.Encoder, v *unique.Handle[string]) error {
				ok := *v != (unique.Handle[string]{})
				if err := nice.MarshalEncode(enc, &ok); err != nil {
					return err
				}
				if ok {
					tmp := v.Value()
					return nice.MarshalEncode(enc, &tmp)
				}
				return nil
			},
			func(dec *nice.Decoder, v *unique.Handle[string]) error {
				var ok bool
				if err := nice.UnmarshalDecode(dec, &ok); err != nil {
					return err
				}
				if ok {
					var tmp string
					if err := nice.UnmarshalDecode(dec, &tmp); err != nil {
						return err
					}
					*v = unique.Make(tmp)
				} else {
					*v = (unique.Handle[string]{})
				}
				return nil
			},
		),
		// nice.MakeInterfaceArshaler2[game.ScriptState](
		// 	func(typ reflect.Type) [32]byte {
		// 		if _, ok := game.Scripts[typ]; !ok {
		// 			panic(fmt.Sprintf("bad %#v", typ))
		// 		}
		// 		return hash(typ.Name())
		// 	},
		// 	func(typId [32]byte) reflect.Type {
		// 		// TODO: cache things in a map
		// 		for typ := range game.Scripts {
		// 			if hash(typ.Name()) == typId {
		// 				return typ
		// 			}
		// 		}
		// 		panic(fmt.Sprintf("bad %v", typId))
		// 	}),
		nice.MakeArshaler[game.ScriptState](
			func(enc *nice.Encoder, v *game.ScriptState) error {
				data := reflect.ValueOf(*v) // TODO: do ValueOf(v).Elem().Elem()

				var id [32]byte
				if *v == nil {
					return nice.MarshalEncode(enc, &id)
				}

				typ := data.Type()

				id = hash(typ.Name())

				if err := nice.MarshalEncode(enc, &id); err != nil {
					return err
				}

				// TODO: any way we could avoid an alloc?
				tmp := reflect.New(typ)
				tmp.Elem().Set(data)
				return nice.MarshalEncode(enc, tmp.Interface())
			},
			func(dec *nice.Decoder, v *game.ScriptState) error {
				var id [32]byte
				if err := nice.UnmarshalDecode(dec, &id); err != nil {
					return err
				}
				if id == ([32]byte{}) {
					*v = nil
					return nil
				}

				typ := func() reflect.Type {
					for typ := range game.Scripts {
						if hash(typ.Name()) == id {
							return typ
						}
					}
					panic(fmt.Sprintf("bad %v", id))
				}()

				// TODO: any way we could avoid an alloc?
				data := reflect.New(typ)
				if err := nice.UnmarshalDecode(dec, data.Interface()); err != nil {
					return err
				}
				*v, _ = reflect.TypeAssert[game.ScriptState](data.Elem())
				return nil
			})))
}

var NiceOptions3 = nice.WithArshalers(nice.MakeInterfaceArshaler[game.InputCmd](game.InputCmdTypes...))

func hash(s string) [32]byte {
	hasher := sha256.New()
	io.WriteString(hasher, s)
	return [32]byte(hasher.Sum(nil))
}
