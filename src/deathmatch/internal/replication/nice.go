package replication

import (
	"worldspawn/deathmatch/internal/game"
	"worldspawn/internal/nice"
)

// TODO: can we get rid of nice in favor of something else, like idk, schemaless
// protobuf?

var NiceOptions = nice.JoinOptions(
	nice.WithMarshaler(game.InterfaceNiceMarshaler[game.InputCmd](game.InputCmds...)),
	nice.WithUnmarshaler(game.InterfaceNiceUnmarshaler[game.InputCmd](game.InputCmds...)),
	nice.WithMarshaler(game.InterfaceNiceMarshaler[game.Entity](game.EntityTypes...)),
	nice.WithUnmarshaler(game.InterfaceNiceUnmarshaler[game.Entity](game.EntityTypes...)))
