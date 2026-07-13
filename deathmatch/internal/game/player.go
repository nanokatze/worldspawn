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
// TODO: We need to think how to handle the case when Camera is independent of
// player input (e.g. when we're flying along some track.) In that case, the
// client should not set T0 and T1 to whatever value we return
func (world *World) Camera(playerID ecs.ID) ecs.ID {
	// TODO: it should really poke the script Camera func probably

	player := world.GetEntity2(playerID)
	if !player.Valid() {
		return 0
	}
	playerState, ok := player.ScriptState().(Player)
	if !ok {
		return 0
	}

	pawn := world.GetEntity2(playerState.Pawn)
	if !pawn.Valid() {
		return 0
	}
	pawnState, ok := pawn.ScriptState().(Gladiator) // TODO: could we just poke a script function on the entity?
	if !ok {
		return 0
	}

	return pawnState.FirstPersonCamera
}

// TODO: think of a good way to implement HUD and the GUI. I suppose what we
// could do is have a pile of entities that would have a Script func that would
// spit out ops into gio-esque ops sequence.
func (world *World) Overlay(playerID ecs.ID, health, bleed *int32) {
	player := world.GetEntity2(playerID)
	if !player.Valid() {
		return
	}
	playerState, ok := player.ScriptState().(Player)
	if !ok {
		return
	}

	pawn := world.GetEntity2(playerState.Pawn)
	if !pawn.Valid() {
		return
	}
	pawnState, ok := pawn.ScriptState().(Gladiator) // TODO: could we just poke a script function on the entity?
	if !ok {
		return
	}

	*health = pawnState.Vitals.Health
	*bleed = pawnState.Vitals.HealthToBleed
}

// TODO: this should not take UpdateParams *at all* I think. Actually we still
// need flags, but not Δt.
func (world *World) HandleInput(playerID ecs.ID, cmd TimestampedInputCmd, info *UpdateParams) {
	playerState := mustOk(world.GetEntity[Player](playerID))
	if pawn := world.GetEntity2(playerState.Pawn); pawn.Valid() {
		pawn.Script().Input(info, world, pawn.ID(), cmd)
	} else {
		if !info.Speculating {
			spawnPoint := world.findSpawnPoint()
			world.logger.Info("spawn", "player", playerID, "T", spawnPoint)

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
