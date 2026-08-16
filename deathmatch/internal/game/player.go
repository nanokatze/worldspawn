package game

import (
	"reflect"
)

type Player struct {
	ID string

	Loadout struct{}

	Score struct {
		Kills  int32
		Deaths int32
	}

	Pawn EntityID
}

func init() {
	Scripts[reflect.TypeFor[Player]()] = script{}
}

// TODO: should be called something like CreatePlayer or something. We only
// really need to poke this when new client joins. We could also rename Player
// to Client or Connection or User or idk.
// TODO: see if we can somehow defer things with updates, so that as much stuff is kept in parallel
func (world *World) SpawnPlayer(info *UpdateParams) EntityID {
	player := world.CreateEntity(info)
	player.SetScriptState(Player{})
	return player.ID()
}

// TODO: returning EntityID is kinda meh, ideally we'd return a pile of data that
// can be fed straight into pathtracer
// TODO: We need to think how to handle the case when Camera is independent of
// player input (e.g. when we're flying along some track.) In that case, the
// client should not set T0 and T1 to whatever value we return
// TODO: delegate this stuff to script somehow? So that it would work like
// HandleInput.
func (world *World) Camera(playerID EntityID) EntityID {
	player := world.Entity(playerID)
	if !player.IsValid() {
		return 0
	}
	playerState, ok := player.ScriptState().(Player)
	if !ok {
		return 0
	}

	pawn := world.Entity(playerState.Pawn)
	if !pawn.IsValid() {
		return 0
	}
	pawnState, ok := pawn.ScriptState().(Gladiator) // TODO: could we just poke a script function on the entity?
	if !ok {
		return 0
	}

	return pawnState.Head
}

// TODO: this should not take UpdateParams *at all* I think. Actually we still
// need flags, but not Δt.
func (world *World) HandleInput(playerID EntityID, cmd TimestampedInputCmd, info *UpdateParams) {
	playerState := world.Entity(playerID).ScriptState().(Player)
	if pawn := world.Entity(playerState.Pawn); pawn.IsValid() {
		pawn.Script().Input(info, pawn.ID(), world, cmd)
	} else {
		if !info.Speculating {
			spawnPoint := world.findPlayerSpawn()
			world.logger.Info("spawn", "player", playerID, "T", spawnPoint)

			playerState.Pawn = world.spawnGladiator(spawnPoint, info)
			world.Entity(playerID).SetScriptState(playerState)
		}
	}
}
