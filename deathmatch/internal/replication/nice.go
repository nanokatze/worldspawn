package replication

import (
	"worldspawn/deathmatch/internal/game"
	"worldspawn/internal/nice"
)

var NiceOptions = nice.WithArshalers(nice.JoinArshalers(
	nice.MakeUniqueHandleArshaler[string](),
	// TODO: we'll want to kill this if scripts are going to be dynamically
	// loaded
	nice.MakeInterfaceArshaler[game.Entity](game.EntityTypes...),
	nice.MakeInterfaceArshaler[game.InputCmd](game.InputCmdTypes...)))
