package replication

import (
	"worldspawn/deathmatch/internal/game"
	"worldspawn/internal/nice"
)

var NiceOptions = nice.WithArshalers(nice.JoinArshalers(
	nice.InterfaceArshaler[game.Entity](game.EntityTypes...),
	nice.InterfaceArshaler[game.InputCmd](game.InputCmdTypes...)))
