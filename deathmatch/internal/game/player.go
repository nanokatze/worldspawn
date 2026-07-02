package game

import (
	"worldspawn/internal/ecs"
)

type Player struct {
	ControlledCharacter ecs.ID

	Score struct {
		Kills  int32
		Deaths int32
	}

	Loadout struct {
	}
}

func (Player) entity() {}

// TODO: returning ecs.ID is kinda meh, ideally we'd return a pile of data that
// can be fed straight into pathtracer
// TODO: make this a method on the World? We need to think how to handle the
// case when Camera is independent of player input (e.g. when we're flying along
// some track.) In that case, the client should not set T0 and T1 to whatever value we return
func (player Player) Camera(world *World) ecs.ID {
	char, ok := world.GetEntity[Gladiator](player.ControlledCharacter)
	if !ok {
		return 0
	}
	return char.FirstPersonCamera
}

func (world *World) HandleInput(player ecs.ID, cmd TimestampedInputCmd, info *UpdateParams) {
	playerState := mustOk(world.GetEntity[Player](player))
	if world.EntityExists(playerState.ControlledCharacter) {
		world.GetScriptFuncs(playerState.ControlledCharacter).
			Input(info, world, playerState.ControlledCharacter, cmd)
	} else {
		if !info.Speculating {
			spawnPoint := world.findSpawnPoint()
			info.Logger.Info("spawn", "player", player, "T", spawnPoint)

			playerState.ControlledCharacter = world.spawnGladiator(spawnPoint, info)
			world.Entity.Set(player, playerState)
		}
	}
}

// TODO: should be called something like CreatePlayer or something. We only
// really need to poke this when new client joins. We could also rename Player
// to Client or Connection or User or idk.
func (world *World) SpawnPlayer(info *UpdateParams) ecs.ID {
	player := world.CreateEntity(info)
	world.Entity.Set(player, Player{})
	return player
}
