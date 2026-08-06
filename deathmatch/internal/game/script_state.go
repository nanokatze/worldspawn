package game

type ScriptState interface{}

func (e Entity) ScriptState() ScriptState { return e.world.ScriptState.Load(e.id.Index()) }

func (e Entity) SetScriptState(v ScriptState) { e.world.ScriptState.Store(e.id.Index(), v) }
