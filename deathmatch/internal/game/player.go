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
func (player Player) Camera(w *Scene) ecs.ID {
	if player.ControlledCharacter == 0 {
		return 0 // TODO: get rid of this and make the callers care
	}
	char := mustOk(SceneGetEntity[Gladiator](w, player.ControlledCharacter))
	return char.FirstPersonCamera
}

func (w *Scene) HandleInput(playerID ecs.ID, cmd TimestampedInputCmd, info *UpdateParams) {
	player := mustOk(SceneGetEntity[Player](w, playerID))
	char := mustOk(SceneGetEntity[PlayableCharacter](w, player.ControlledCharacter))
	char.HandleInput(w, player.ControlledCharacter, cmd, info)
}

// TODO: this should be a method on the scene
func SpawnPlayer(scene *Scene, info *UpdateParams) ecs.ID {
	player := scene.CreateEntity(info)

	char := createGladiator(scene, info)
	scene.Entity.Set(player, Player{
		ControlledCharacter: char,
	})

	return player
}
