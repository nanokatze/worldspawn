package replication

import (
	"worldspawn/deathmatch/internal/game"
	"worldspawn/internal/nice"
)

var NiceOptions = nice.WithArshalers(nice.JoinArshalers(
	nice.InterfaceArshaler[game.InputCmd](game.InputCmds...),
	nice.InterfaceArshaler[game.Entity](game.EntityTypes...)))
