package game

import (
	"worldspawn/internal/ecs"
)

type Player struct {
	Pawn ecs.ID

	Score struct {
		Kills  int32
		Deaths int32
	}

	Loadout struct {
	}
}

// TODO: returning ecs.ID is kinda meh, ideally we'd return a pile of data that
// can be fed straight into pathtracer
// TODO: make this a method on the World? We need to think how to handle the
// case when Camera is independent of player input (e.g. when we're flying along
// some track.) In that case, the client should not set T0 and T1 to whatever value we return
func (player Player) Camera(world *World) ecs.ID {
	char, ok := world.GetEntity[Gladiator](player.Pawn)
	if !ok {
		return 0
	}
	return char.FirstPersonCamera
}

func (world *World) HandleInput(playerID ecs.ID, cmd TimestampedInputCmd, info *UpdateParams) {
	playerState := mustOk(world.GetEntity[Player](playerID))
	if pawn := world.GetEntity2(playerState.Pawn); pawn.Valid() {
		pawn.Script().Input(info, world, pawn.ID(), cmd)
	} else {
		if !info.Speculating {
			spawnPoint := world.findSpawnPoint()
			info.Logger.Info("spawn", "player", playerID, "T", spawnPoint)

			playerState.Pawn = world.spawnGladiator(spawnPoint, info)
			world.Entity.Set(playerID, playerState)
		}
	}
}

// TODO: should be called something like CreatePlayer or something. We only
// really need to poke this when new client joins. We could also rename Player
// to Client or Connection or User or idk.
func (world *World) SpawnPlayer(info *UpdateParams) ecs.ID {
	player := world.CreateEntity(info)
	player.SetScriptState(Player{})
	return player.ID()
}
