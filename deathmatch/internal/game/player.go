package game

import (
	"worldspawn/internal/ecs"
)

type Player struct {
	Score struct {
		Kills  int32
		Deaths int32
	}

	ControlledCharacter ecs.ID
}

func (Player) entity() {}

// TODO: delete, client should figure this out by itself
func (player Player) Camera(world *World) ecs.ID {
	if player.ControlledCharacter == 0 {
		return 0 // TODO: get rid of this and make the callers care
	}
	char := mustOk(SceneGetEntity[Gladiator](world, player.ControlledCharacter))
	return char.FirstPersonCamera
}

func (world *World) HandleInput(playerID ecs.ID, cmd TimestampedInputCmd, info *UpdateParams) {
	player := mustOk(SceneGetEntity[Player](world, playerID))
	char := mustOk(SceneGetEntity[PlayableCharacter](world, player.ControlledCharacter))
	char.HandleInput(world, player.ControlledCharacter, cmd, info)
}

func (world *World) SpawnPlayer(info *UpdateParams) ecs.ID {
	player := world.CreateEntity(info)

	char := world.spawnGladiator(world.findSpawnPoint(), info)
	world.Entity.Set(player, Player{
		ControlledCharacter: char,
	})

	return player
}
