package worldspawn

import (
	"fmt"
	"io"
	"reflect"

	"worldspawn/internal/nice"

	"golang.org/x/crypto/blake2b"
)

// TODO: give these a type
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

	ActionMoveX
	ActionMoveY

	ActionDLookX
	ActionDLookY
)

// TODO: we need a tracker object so that we can normalize value per action and
// filter things out
func AppendAction(dst []TimestampedInputCmd, time Time, action int, value float32) []TimestampedInputCmd {
	cmd := actionToInputCmd(action, value)
	if cmd != nil {
		dst = append(dst, TimestampedInputCmd{Time: time, Cmd: cmd})
	}
	return dst
}

// TODO: with some extra effort we can make InputCmd values private
func actionToInputCmd(action int, value float32) InputCmd {
	switch action {
	case ActionJump, ActionCrouch, ActionAttack, ActionReload:
		if value != 0 {
			return ButtonDown(action)
		} else {
			return ButtonUp(action)
		}

	case ActionSlot0, ActionSlot1, ActionSlot2, ActionSlot3:
		// TODO: we should do nothing if value == 0
		return Slot(action - ActionSlot0)

	case ActionMoveX:
		return MoveX(value)

	case ActionMoveY:
		return MoveY(value)

	case ActionDLookX:
		return DLookX(value)

	case ActionDLookY:
		return DLookY(value)

	default:
		panic("unknown action")
	}
}

// TODO: serialize input cmds to []byte immediately, rather than pass structs
// around?

type TimestampedInputCmd struct {
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

// TODO: delete this
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

type DLookX float32
type DLookY float32

type MoveX float32
type MoveY float32

// TODO: replace generic ButtonDown and ButtonUp with a definition per each
// button and a
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
