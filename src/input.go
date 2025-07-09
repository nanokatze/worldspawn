package worldspawn

import (
	"fmt"
	"io"
	"reflect"

	"worldspawn/experiments/encoding/nice"

	"golang.org/x/crypto/blake2b"
)

// TODO: clean this up please

const (
	_ = iota
	ActionJump
	ActionCrouch
	ActionAttack
	ActionReload
	ActionSlot0
	ActionSlot1
	ActionSlot2
	ActionSlot3
	// Should analog actions come before digital, or after, like here? If we put
	// them before, the enums can match up with buttons.
	ActionMoveX
	ActionMoveY
	ActionDLookX
	ActionDLookY
)

// TODO: should also specify entity this input command is for? Or perhaps it
// should be delegated to the client and server to handle that.
// TODO: rename. InputCmd{Decorated,Timestamped}?
type InputCmd2 struct {
	Time Time
	Cmd  InputCmd
}

type InputCmd any

func init() {
	registerInputCommand[DLookX]()
	registerInputCommand[DLookY]()
	registerInputCommand[MoveX]()
	registerInputCommand[MoveY]()
	registerInputCommand[ButtonDown]()
	registerInputCommand[ButtonUp]()
	registerInputCommand[Slot]()
}

type arshaltab struct {
	typ map[reflect.Type]struct {
		name string
		hash [4]byte // TODO: input cmds would benefit from 2 or even 1 byte identifiers
	}
	name map[string]reflect.Type
	hash map[[4]byte]reflect.Type
}

func (a *arshaltab) Register(name string, t reflect.Type) {
	if _, ok := a.name[name]; ok {
		panic("collision")
	}

	h, _ := blake2b.New(4, nil)
	h.Write([]byte(name))

	hash := [4]byte(h.Sum(nil))

	if _, ok := a.hash[hash]; ok {
		panic("collision")
	}

	a.typ[t] = struct {
		name string
		hash [4]byte
	}{
		name: name,
		hash: hash,
	}
	a.name[name] = t
	a.hash[hash] = t
}

func mkarshaltab() *arshaltab {
	return &arshaltab{
		typ: map[reflect.Type]struct {
			name string
			hash [4]byte
		}{},
		name: map[string]reflect.Type{},
		hash: map[[4]byte]reflect.Type{},
	}
}

var inputcmdarshaltab = mkarshaltab()

func registerInputCommand[T any]() {
	t := reflect.TypeFor[T]()
	inputcmdarshaltab.Register(t.Name(), t)
}

// TODO: when we move game stuff into game/???/, arshaler code should not be in
// the game's code

func InputCommandNiceMarshal(enc *nice.Encoder, icmd *InputCmd) error {
	data := reflect.ValueOf(*icmd)
	typ := data.Type()

	info, ok := inputcmdarshaltab.typ[typ]
	if !ok {
		panic("bad")
	}

	if _, err := enc.Writer().Write(info.hash[:]); err != nil {
		return err
	}

	// TODO: any way we could avoid an alloc?
	tmp := reflect.New(typ)
	tmp.Elem().Set(data)
	return nice.MarshalEncode(enc, tmp.Interface())
}

func InputCommandNiceUnmarshal(dec *nice.Decoder, icmd *InputCmd) error {
	buf := dec.Scratch(4)
	if _, err := io.ReadFull(dec.Reader(), buf); err != nil {
		return err
	}
	hash := [4]byte(buf)

	typ, ok := inputcmdarshaltab.hash[hash]
	if !ok {
		return fmt.Errorf("unknown input command")
	}

	// TODO: any way we could avoid an alloc?
	data := reflect.New(typ)
	if err := nice.UnmarshalDecode(dec, data.Interface()); err != nil {
		return err
	}
	*icmd = data.Elem().Interface()
	return nil
}

// TODO: use SNORM for move and look?

// TODO: prefix these with InputCmd probably

// TODO: collapse these into the same command
type DLookX float32
type DLookY float32

// TODO: collapse these into the same command
type MoveX float32
type MoveY float32

type Button uint8

const (
	_ Button = iota
	ButtonJump
	ButtonCrouch
	ButtonAttack
	ButtonReload
)

type ButtonDown Button
type ButtonUp Button

// TODO: rename this into something more descriptive (e.g. UseWeaponInSlot).
// Also, we actually want to use weapon by its ID, and slots should be
// user-configurable probably. Actually, this would prevent user from having
// multiple instances of the same weapon, so we need to rethink that. I guess we
// should call these things "slots" then? TODO: think harder
//
// TODO: remove in favor of slot buttons, so that weapon switching is entirely
// dictated by the game.
type Slot int8
