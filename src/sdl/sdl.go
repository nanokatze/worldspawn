// Minimal SDL Go bindings that only expose stuff we need.
package sdl

// ALWAYS use SDL from the environment and dynamically link it

// #cgo pkg-config: sdl3
//
// #include <stdlib.h>
//
// #include <SDL3/SDL.h>
import "C"

import (
	"errors"
	"reflect"
	"runtime"
	"structs"
	"unsafe"
)

// Init

type InitFlags uint32

const (
	INIT_AUDIO    InitFlags = C.SDL_INIT_AUDIO
	INIT_VIDEO    InitFlags = C.SDL_INIT_VIDEO
	INIT_JOYSTICK InitFlags = C.SDL_INIT_JOYSTICK
	INIT_HAPTIC   InitFlags = C.SDL_INIT_HAPTIC
	INIT_GAMEPAD  InitFlags = C.SDL_INIT_GAMEPAD
	INIT_EVENTS   InitFlags = C.SDL_INIT_EVENTS
	INIT_SENSOR   InitFlags = C.SDL_INIT_SENSOR
)

func InitSubSystem(flags InitFlags) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !C.SDL_InitSubSystem(C.SDL_InitFlags(flags)) {
		return getError()
	}
	return nil
}

// Hints

func SetHint(name, value string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !C.SDL_SetHint(cstring(name), cstring(value)) {
		return getError()
	}
	return nil
}

// Properties

type PropertiesID uint32

func CreateProperties() (PropertiesID, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	props := PropertiesID(C.SDL_CreateProperties())
	if props == 0 {
		return 0, getError()
	}
	return props, nil
}

func (props PropertiesID) Destroy() {
	C.SDL_DestroyProperties(C.SDL_PropertiesID(props))
}

func (props PropertiesID) Pointer(prop string, value uintptr) uintptr {
	// TODO: wire-in the default value
	return uintptr(C.SDL_GetPointerProperty(C.SDL_PropertiesID(props), cstring(prop), nil))
}

func (props PropertiesID) SetString(prop string, value string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !C.SDL_SetStringProperty(C.SDL_PropertiesID(props), cstring(prop), cstring(value)) {
		return getError()
	}
	return nil
}

func (props PropertiesID) SetNumber(prop string, value int64) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !C.SDL_SetNumberProperty(C.SDL_PropertiesID(props), cstring(prop), C.Sint64(value)) {
		return getError()
	}
	return nil
}

func (props PropertiesID) SetBoolean(prop string, value bool) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !C.SDL_SetBooleanProperty(C.SDL_PropertiesID(props), cstring(prop), C.bool(value)) {
		return getError()
	}
	return nil
}

func WithStringProperty(prop string, value string) func(props PropertiesID) error {
	return func(props PropertiesID) error { return props.SetString(prop, value) }
}

func WithNumberProperty(prop string, value int64) func(props PropertiesID) error {
	return func(props PropertiesID) error { return props.SetNumber(prop, value) }
}

func WithBooleanProperty(prop string, value bool) func(props PropertiesID) error {
	return func(props PropertiesID) error { return props.SetBoolean(prop, value) }
}

// Events

//go:generate stringer -type EventType -trimprefix EVENT_

type EventType uint32

const (
	EVENT_QUIT EventType = C.SDL_EVENT_QUIT

	EVENT_WINDOW_PIXEL_SIZE_CHANGED EventType = C.SDL_EVENT_WINDOW_PIXEL_SIZE_CHANGED

	EVENT_KEY_DOWN EventType = C.SDL_EVENT_KEY_DOWN
	EVENT_KEY_UP   EventType = C.SDL_EVENT_KEY_UP

	EVENT_MOUSE_MOTION      EventType = C.SDL_EVENT_MOUSE_MOTION
	EVENT_MOUSE_BUTTON_DOWN EventType = C.SDL_EVENT_MOUSE_BUTTON_DOWN
	EVENT_MOUSE_BUTTON_UP   EventType = C.SDL_EVENT_MOUSE_BUTTON_UP
	EVENT_MOUSE_WHEEL       EventType = C.SDL_EVENT_MOUSE_WHEEL

	EVENT_GAMEPAD_AXIS_MOTION          EventType = C.SDL_EVENT_GAMEPAD_AXIS_MOTION
	EVENT_GAMEPAD_BUTTON_DOWN          EventType = C.SDL_EVENT_GAMEPAD_BUTTON_DOWN
	EVENT_GAMEPAD_BUTTON_UP            EventType = C.SDL_EVENT_GAMEPAD_BUTTON_UP
	EVENT_GAMEPAD_ADDED                EventType = C.SDL_EVENT_GAMEPAD_ADDED
	EVENT_GAMEPAD_REMOVED              EventType = C.SDL_EVENT_GAMEPAD_REMOVED
	EVENT_GAMEPAD_REMAPPED             EventType = C.SDL_EVENT_GAMEPAD_REMAPPED
	EVENT_GAMEPAD_TOUCHPAD_DOWN        EventType = C.SDL_EVENT_GAMEPAD_TOUCHPAD_DOWN
	EVENT_GAMEPAD_TOUCHPAD_MOTION      EventType = C.SDL_EVENT_GAMEPAD_TOUCHPAD_MOTION
	EVENT_GAMEPAD_TOUCHPAD_UP          EventType = C.SDL_EVENT_GAMEPAD_TOUCHPAD_UP
	EVENT_GAMEPAD_SENSOR_UPDATE        EventType = C.SDL_EVENT_GAMEPAD_SENSOR_UPDATE
	EVENT_GAMEPAD_UPDATE_COMPLETE      EventType = C.SDL_EVENT_GAMEPAD_UPDATE_COMPLETE
	EVENT_GAMEPAD_STEAM_HANDLE_UPDATED EventType = C.SDL_EVENT_GAMEPAD_STEAM_HANDLE_UPDATED
)

// TODO: rename so that it's clear it's not SDL_Event? E.g. EventPointer or idk.
type Event interface{ event() }

type QuitEvent struct {
	_         structs.HostLayout
	Type      EventType
	_         uint32
	Timestamp uint64
}

func (*QuitEvent) event() {}

type WindowEvent struct {
	_         structs.HostLayout
	Type      EventType
	_         uint32
	Timestamp uint64
	WindowID  WindowID
	Data1     int32
	Data2     int32
}

type WindowPixelSizeChangedEvent WindowEvent

func (*WindowPixelSizeChangedEvent) event() {}

type KeyboardEvent struct {
	_         structs.HostLayout
	Type      EventType
	_         uint32
	Timestamp uint64
	WindowID  WindowID
	Which     KeyboardID
	Scancode  Scancode
	Key       Keycode
	Mod       Keymod
	Raw       uint16
	Down      bool
	Repeat    bool
}

type KeyDownEvent KeyboardEvent
type KeyUpEvent KeyboardEvent

func (*KeyDownEvent) event() {}
func (*KeyUpEvent) event()   {}

type MouseMotionEvent struct {
	_         structs.HostLayout
	Type      EventType
	_         uint32
	Timestamp uint64
	WindowID  WindowID
	Which     MouseID
	State     MouseButtonFlags
	X         float32
	Y         float32
	XRel      float32
	YRel      float32
}

func (*MouseMotionEvent) event() {}

type MouseButtonEvent struct {
	_         structs.HostLayout
	Type      EventType
	_         uint32
	Timestamp uint64
	WindowID  WindowID
	Which     MouseID
	Button    uint8
	Down      bool
	Clicks    uint8
	_         uint8
	X         float32
	Y         float32
}

type (
	MouseButtonDownEvent MouseButtonEvent
	MouseButtonUpEvent   MouseButtonEvent
)

func (*MouseButtonDownEvent) event() {}
func (*MouseButtonUpEvent) event()   {}

type GamepadAxisEvent struct {
	_         structs.HostLayout
	Type      EventType
	_         uint32
	Timestamp uint64
	Which     JoystickID
	Axis      GamepadAxis
	_         uint8
	_         uint8
	_         uint8
	Value     int16
	_         uint16
}

type GamepadAxisMotionEvent GamepadAxisEvent

func (*GamepadAxisMotionEvent) event() {}

type GamepadButtonEvent struct {
	_         structs.HostLayout
	Type      EventType
	_         uint32
	Timestamp uint64
	Which     JoystickID
	Button    uint8
	Down      bool
	_         uint8
	_         uint8
}

type (
	GamepadButtonDownEvent GamepadButtonEvent
	GamepadButtonUpEvent   GamepadButtonEvent
)

func (*GamepadButtonDownEvent) event() {}
func (*GamepadButtonUpEvent) event()   {}

type GamepadDeviceEvent struct {
	_         structs.HostLayout
	Type      EventType
	_         uint32
	Timestamp uint64
	Which     JoystickID
}

type (
	GamepadAddedEvent          GamepadDeviceEvent
	GamepadRemappedEvent       GamepadDeviceEvent
	GamepadRemovedEvent        GamepadDeviceEvent
	GamepadUpdateCompleteEvent GamepadDeviceEvent
)

func (*GamepadAddedEvent) event()          {}
func (*GamepadRemappedEvent) event()       {}
func (*GamepadRemovedEvent) event()        {}
func (*GamepadUpdateCompleteEvent) event() {}

func _() {
	var x [1]int
	_ = x[unsafe.Sizeof(QuitEvent{})-unsafe.Sizeof(C.SDL_QuitEvent{})]
	_ = x[unsafe.Sizeof(WindowEvent{})-unsafe.Sizeof(C.SDL_WindowEvent{})]
	_ = x[unsafe.Sizeof(KeyboardEvent{})-unsafe.Sizeof(C.SDL_KeyboardEvent{})]
	_ = x[unsafe.Sizeof(MouseMotionEvent{})-unsafe.Sizeof(C.SDL_MouseMotionEvent{})]
	_ = x[unsafe.Sizeof(MouseButtonEvent{})-unsafe.Sizeof(C.SDL_MouseButtonEvent{})]
	_ = x[unsafe.Sizeof(GamepadAxisEvent{})-unsafe.Sizeof(C.SDL_GamepadAxisEvent{})]
	_ = x[unsafe.Sizeof(GamepadButtonEvent{})-unsafe.Sizeof(C.SDL_GamepadButtonEvent{})]
	_ = x[unsafe.Sizeof(GamepadDeviceEvent{})-unsafe.Sizeof(C.SDL_GamepadDeviceEvent{})]
}

var eventTypes = map[EventType]reflect.Type{
	EVENT_QUIT:                      reflect.TypeFor[*QuitEvent](),
	EVENT_WINDOW_PIXEL_SIZE_CHANGED: reflect.TypeFor[*WindowPixelSizeChangedEvent](),
	EVENT_KEY_DOWN:                  reflect.TypeFor[*KeyDownEvent](),
	EVENT_KEY_UP:                    reflect.TypeFor[*KeyUpEvent](),
	EVENT_MOUSE_MOTION:              reflect.TypeFor[*MouseMotionEvent](),
	EVENT_MOUSE_BUTTON_DOWN:         reflect.TypeFor[*MouseButtonDownEvent](),
	EVENT_MOUSE_BUTTON_UP:           reflect.TypeFor[*MouseButtonUpEvent](),
	EVENT_GAMEPAD_AXIS_MOTION:       reflect.TypeFor[*GamepadAxisMotionEvent](),
	EVENT_GAMEPAD_BUTTON_DOWN:       reflect.TypeFor[*GamepadButtonDownEvent](),
	EVENT_GAMEPAD_BUTTON_UP:         reflect.TypeFor[*GamepadButtonUpEvent](),
	EVENT_GAMEPAD_ADDED:             reflect.TypeFor[*GamepadAddedEvent](),
	EVENT_GAMEPAD_REMAPPED:          reflect.TypeFor[*GamepadRemappedEvent](),
	EVENT_GAMEPAD_REMOVED:           reflect.TypeFor[*GamepadRemovedEvent](),
	EVENT_GAMEPAD_UPDATE_COMPLETE:   reflect.TypeFor[*GamepadUpdateCompleteEvent](),
}

func WaitEvent() (Event, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var event C.SDL_Event
	if !C.SDL_WaitEvent(&event) {
		return nil, getError()
	}
	return translateEvent(&event), nil
}

func translateEvent(p *C.SDL_Event) Event {
	eventType := *(*EventType)(unsafe.Pointer(p))

	typ := eventTypes[eventType]
	if typ == nil {
		return nil
	}

	pTyped, _ := reflect.TypeAssert[Event](reflect.NewAt(typ.Elem(), unsafe.Pointer(p)))
	return pTyped
}

// Video

type WindowID uint32

type Window C.SDL_Window

const (
	PROP_WINDOW_CREATE_HEIGHT_NUMBER                       = C.SDL_PROP_WINDOW_CREATE_HEIGHT_NUMBER
	PROP_WINDOW_CREATE_HIGH_PIXEL_DENSITY_BOOLEAN          = C.SDL_PROP_WINDOW_CREATE_HIGH_PIXEL_DENSITY_BOOLEAN
	PROP_WINDOW_CREATE_RESIZABLE_BOOLEAN                   = C.SDL_PROP_WINDOW_CREATE_RESIZABLE_BOOLEAN
	PROP_WINDOW_CREATE_TITLE_STRING                        = C.SDL_PROP_WINDOW_CREATE_TITLE_STRING
	PROP_WINDOW_CREATE_VULKAN_BOOLEAN                      = C.SDL_PROP_WINDOW_CREATE_VULKAN_BOOLEAN
	PROP_WINDOW_CREATE_WIDTH_NUMBER                        = C.SDL_PROP_WINDOW_CREATE_WIDTH_NUMBER
	PROP_WINDOW_CREATE_WAYLAND_SURFACE_ROLE_CUSTOM_BOOLEAN = C.SDL_PROP_WINDOW_CREATE_WAYLAND_SURFACE_ROLE_CUSTOM_BOOLEAN
)

// TODO: introduce a wrapped CreateWindow with func vararg config with
// properties (i.e. the thing would accept stuff like WithStringProperty,
// WithNumberProperty, etc)

func CreateWindowWithProperties(props PropertiesID) (*Window, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	window := (*Window)(C.SDL_CreateWindowWithProperties(C.SDL_PropertiesID(props)))
	if window == nil {
		return nil, getError()
	}
	return window, nil
}

func (w *Window) Properties() PropertiesID {
	return PropertiesID(C.SDL_GetWindowProperties((*C.SDL_Window)(w)))
}

func (w *Window) DisplayScale() (float32, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	scale := float32(C.SDL_GetWindowDisplayScale((*C.SDL_Window)(w)))
	if scale == 0 {
		return 0, getError()
	}
	return scale, nil
}

// TODO: should be moved elsewhere

// Keyboard

type KeyboardID uint32

// Keycode

//go:generate stringer -type Keycode -trimprefix K_

type Keycode int32

const (
	K_RETURN               Keycode = C.SDLK_RETURN
	K_ESCAPE               Keycode = C.SDLK_ESCAPE
	K_BACKSPACE            Keycode = C.SDLK_BACKSPACE
	K_TAB                  Keycode = C.SDLK_TAB
	K_SPACE                Keycode = C.SDLK_SPACE
	K_EXCLAIM              Keycode = C.SDLK_EXCLAIM
	K_DBLAPOSTROPHE        Keycode = C.SDLK_DBLAPOSTROPHE
	K_HASH                 Keycode = C.SDLK_HASH
	K_DOLLAR               Keycode = C.SDLK_DOLLAR
	K_PERCENT              Keycode = C.SDLK_PERCENT
	K_AMPERSAND            Keycode = C.SDLK_AMPERSAND
	K_APOSTROPHE           Keycode = C.SDLK_APOSTROPHE
	K_LEFTPAREN            Keycode = C.SDLK_LEFTPAREN
	K_RIGHTPAREN           Keycode = C.SDLK_RIGHTPAREN
	K_ASTERISK             Keycode = C.SDLK_ASTERISK
	K_PLUS                 Keycode = C.SDLK_PLUS
	K_COMMA                Keycode = C.SDLK_COMMA
	K_MINUS                Keycode = C.SDLK_MINUS
	K_PERIOD               Keycode = C.SDLK_PERIOD
	K_SLASH                Keycode = C.SDLK_SLASH
	K_0                    Keycode = C.SDLK_0
	K_1                    Keycode = C.SDLK_1
	K_2                    Keycode = C.SDLK_2
	K_3                    Keycode = C.SDLK_3
	K_4                    Keycode = C.SDLK_4
	K_5                    Keycode = C.SDLK_5
	K_6                    Keycode = C.SDLK_6
	K_7                    Keycode = C.SDLK_7
	K_8                    Keycode = C.SDLK_8
	K_9                    Keycode = C.SDLK_9
	K_COLON                Keycode = C.SDLK_COLON
	K_SEMICOLON            Keycode = C.SDLK_SEMICOLON
	K_LESS                 Keycode = C.SDLK_LESS
	K_EQUALS               Keycode = C.SDLK_EQUALS
	K_GREATER              Keycode = C.SDLK_GREATER
	K_QUESTION             Keycode = C.SDLK_QUESTION
	K_AT                   Keycode = C.SDLK_AT
	K_LEFTBRACKET          Keycode = C.SDLK_LEFTBRACKET
	K_BACKSLASH            Keycode = C.SDLK_BACKSLASH
	K_RIGHTBRACKET         Keycode = C.SDLK_RIGHTBRACKET
	K_CARET                Keycode = C.SDLK_CARET
	K_UNDERSCORE           Keycode = C.SDLK_UNDERSCORE
	K_GRAVE                Keycode = C.SDLK_GRAVE
	K_A                    Keycode = C.SDLK_A
	K_B                    Keycode = C.SDLK_B
	K_C                    Keycode = C.SDLK_C
	K_D                    Keycode = C.SDLK_D
	K_E                    Keycode = C.SDLK_E
	K_F                    Keycode = C.SDLK_F
	K_G                    Keycode = C.SDLK_G
	K_H                    Keycode = C.SDLK_H
	K_I                    Keycode = C.SDLK_I
	K_J                    Keycode = C.SDLK_J
	K_K                    Keycode = C.SDLK_K
	K_L                    Keycode = C.SDLK_L
	K_M                    Keycode = C.SDLK_M
	K_N                    Keycode = C.SDLK_N
	K_O                    Keycode = C.SDLK_O
	K_P                    Keycode = C.SDLK_P
	K_Q                    Keycode = C.SDLK_Q
	K_R                    Keycode = C.SDLK_R
	K_S                    Keycode = C.SDLK_S
	K_T                    Keycode = C.SDLK_T
	K_U                    Keycode = C.SDLK_U
	K_V                    Keycode = C.SDLK_V
	K_W                    Keycode = C.SDLK_W
	K_X                    Keycode = C.SDLK_X
	K_Y                    Keycode = C.SDLK_Y
	K_Z                    Keycode = C.SDLK_Z
	K_LEFTBRACE            Keycode = C.SDLK_LEFTBRACE
	K_PIPE                 Keycode = C.SDLK_PIPE
	K_RIGHTBRACE           Keycode = C.SDLK_RIGHTBRACE
	K_TILDE                Keycode = C.SDLK_TILDE
	K_DELETE               Keycode = C.SDLK_DELETE
	K_PLUSMINUS            Keycode = C.SDLK_PLUSMINUS
	K_CAPSLOCK             Keycode = C.SDLK_CAPSLOCK
	K_F1                   Keycode = C.SDLK_F1
	K_F2                   Keycode = C.SDLK_F2
	K_F3                   Keycode = C.SDLK_F3
	K_F4                   Keycode = C.SDLK_F4
	K_F5                   Keycode = C.SDLK_F5
	K_F6                   Keycode = C.SDLK_F6
	K_F7                   Keycode = C.SDLK_F7
	K_F8                   Keycode = C.SDLK_F8
	K_F9                   Keycode = C.SDLK_F9
	K_F10                  Keycode = C.SDLK_F10
	K_F11                  Keycode = C.SDLK_F11
	K_F12                  Keycode = C.SDLK_F12
	K_PRINTSCREEN          Keycode = C.SDLK_PRINTSCREEN
	K_SCROLLLOCK           Keycode = C.SDLK_SCROLLLOCK
	K_PAUSE                Keycode = C.SDLK_PAUSE
	K_INSERT               Keycode = C.SDLK_INSERT
	K_HOME                 Keycode = C.SDLK_HOME
	K_PAGEUP               Keycode = C.SDLK_PAGEUP
	K_END                  Keycode = C.SDLK_END
	K_PAGEDOWN             Keycode = C.SDLK_PAGEDOWN
	K_RIGHT                Keycode = C.SDLK_RIGHT
	K_LEFT                 Keycode = C.SDLK_LEFT
	K_DOWN                 Keycode = C.SDLK_DOWN
	K_UP                   Keycode = C.SDLK_UP
	K_NUMLOCKCLEAR         Keycode = C.SDLK_NUMLOCKCLEAR
	K_KP_DIVIDE            Keycode = C.SDLK_KP_DIVIDE
	K_KP_MULTIPLY          Keycode = C.SDLK_KP_MULTIPLY
	K_KP_MINUS             Keycode = C.SDLK_KP_MINUS
	K_KP_PLUS              Keycode = C.SDLK_KP_PLUS
	K_KP_ENTER             Keycode = C.SDLK_KP_ENTER
	K_KP_1                 Keycode = C.SDLK_KP_1
	K_KP_2                 Keycode = C.SDLK_KP_2
	K_KP_3                 Keycode = C.SDLK_KP_3
	K_KP_4                 Keycode = C.SDLK_KP_4
	K_KP_5                 Keycode = C.SDLK_KP_5
	K_KP_6                 Keycode = C.SDLK_KP_6
	K_KP_7                 Keycode = C.SDLK_KP_7
	K_KP_8                 Keycode = C.SDLK_KP_8
	K_KP_9                 Keycode = C.SDLK_KP_9
	K_KP_0                 Keycode = C.SDLK_KP_0
	K_KP_PERIOD            Keycode = C.SDLK_KP_PERIOD
	K_APPLICATION          Keycode = C.SDLK_APPLICATION
	K_POWER                Keycode = C.SDLK_POWER
	K_KP_EQUALS            Keycode = C.SDLK_KP_EQUALS
	K_F13                  Keycode = C.SDLK_F13
	K_F14                  Keycode = C.SDLK_F14
	K_F15                  Keycode = C.SDLK_F15
	K_F16                  Keycode = C.SDLK_F16
	K_F17                  Keycode = C.SDLK_F17
	K_F18                  Keycode = C.SDLK_F18
	K_F19                  Keycode = C.SDLK_F19
	K_F20                  Keycode = C.SDLK_F20
	K_F21                  Keycode = C.SDLK_F21
	K_F22                  Keycode = C.SDLK_F22
	K_F23                  Keycode = C.SDLK_F23
	K_F24                  Keycode = C.SDLK_F24
	K_EXECUTE              Keycode = C.SDLK_EXECUTE
	K_HELP                 Keycode = C.SDLK_HELP
	K_MENU                 Keycode = C.SDLK_MENU
	K_SELECT               Keycode = C.SDLK_SELECT
	K_STOP                 Keycode = C.SDLK_STOP
	K_AGAIN                Keycode = C.SDLK_AGAIN
	K_UNDO                 Keycode = C.SDLK_UNDO
	K_CUT                  Keycode = C.SDLK_CUT
	K_COPY                 Keycode = C.SDLK_COPY
	K_PASTE                Keycode = C.SDLK_PASTE
	K_FIND                 Keycode = C.SDLK_FIND
	K_MUTE                 Keycode = C.SDLK_MUTE
	K_VOLUMEUP             Keycode = C.SDLK_VOLUMEUP
	K_VOLUMEDOWN           Keycode = C.SDLK_VOLUMEDOWN
	K_KP_COMMA             Keycode = C.SDLK_KP_COMMA
	K_KP_EQUALSAS400       Keycode = C.SDLK_KP_EQUALSAS400
	K_ALTERASE             Keycode = C.SDLK_ALTERASE
	K_SYSREQ               Keycode = C.SDLK_SYSREQ
	K_CANCEL               Keycode = C.SDLK_CANCEL
	K_CLEAR                Keycode = C.SDLK_CLEAR
	K_PRIOR                Keycode = C.SDLK_PRIOR
	K_RETURN2              Keycode = C.SDLK_RETURN2
	K_SEPARATOR            Keycode = C.SDLK_SEPARATOR
	K_OUT                  Keycode = C.SDLK_OUT
	K_OPER                 Keycode = C.SDLK_OPER
	K_CLEARAGAIN           Keycode = C.SDLK_CLEARAGAIN
	K_CRSEL                Keycode = C.SDLK_CRSEL
	K_EXSEL                Keycode = C.SDLK_EXSEL
	K_KP_00                Keycode = C.SDLK_KP_00
	K_KP_000               Keycode = C.SDLK_KP_000
	K_THOUSANDSSEPARATOR   Keycode = C.SDLK_THOUSANDSSEPARATOR
	K_DECIMALSEPARATOR     Keycode = C.SDLK_DECIMALSEPARATOR
	K_CURRENCYUNIT         Keycode = C.SDLK_CURRENCYUNIT
	K_CURRENCYSUBUNIT      Keycode = C.SDLK_CURRENCYSUBUNIT
	K_KP_LEFTPAREN         Keycode = C.SDLK_KP_LEFTPAREN
	K_KP_RIGHTPAREN        Keycode = C.SDLK_KP_RIGHTPAREN
	K_KP_LEFTBRACE         Keycode = C.SDLK_KP_LEFTBRACE
	K_KP_RIGHTBRACE        Keycode = C.SDLK_KP_RIGHTBRACE
	K_KP_TAB               Keycode = C.SDLK_KP_TAB
	K_KP_BACKSPACE         Keycode = C.SDLK_KP_BACKSPACE
	K_KP_A                 Keycode = C.SDLK_KP_A
	K_KP_B                 Keycode = C.SDLK_KP_B
	K_KP_C                 Keycode = C.SDLK_KP_C
	K_KP_D                 Keycode = C.SDLK_KP_D
	K_KP_E                 Keycode = C.SDLK_KP_E
	K_KP_F                 Keycode = C.SDLK_KP_F
	K_KP_XOR               Keycode = C.SDLK_KP_XOR
	K_KP_POWER             Keycode = C.SDLK_KP_POWER
	K_KP_PERCENT           Keycode = C.SDLK_KP_PERCENT
	K_KP_LESS              Keycode = C.SDLK_KP_LESS
	K_KP_GREATER           Keycode = C.SDLK_KP_GREATER
	K_KP_AMPERSAND         Keycode = C.SDLK_KP_AMPERSAND
	K_KP_DBLAMPERSAND      Keycode = C.SDLK_KP_DBLAMPERSAND
	K_KP_VERTICALBAR       Keycode = C.SDLK_KP_VERTICALBAR
	K_KP_DBLVERTICALBAR    Keycode = C.SDLK_KP_DBLVERTICALBAR
	K_KP_COLON             Keycode = C.SDLK_KP_COLON
	K_KP_HASH              Keycode = C.SDLK_KP_HASH
	K_KP_SPACE             Keycode = C.SDLK_KP_SPACE
	K_KP_AT                Keycode = C.SDLK_KP_AT
	K_KP_EXCLAM            Keycode = C.SDLK_KP_EXCLAM
	K_KP_MEMSTORE          Keycode = C.SDLK_KP_MEMSTORE
	K_KP_MEMRECALL         Keycode = C.SDLK_KP_MEMRECALL
	K_KP_MEMCLEAR          Keycode = C.SDLK_KP_MEMCLEAR
	K_KP_MEMADD            Keycode = C.SDLK_KP_MEMADD
	K_KP_MEMSUBTRACT       Keycode = C.SDLK_KP_MEMSUBTRACT
	K_KP_MEMMULTIPLY       Keycode = C.SDLK_KP_MEMMULTIPLY
	K_KP_MEMDIVIDE         Keycode = C.SDLK_KP_MEMDIVIDE
	K_KP_PLUSMINUS         Keycode = C.SDLK_KP_PLUSMINUS
	K_KP_CLEAR             Keycode = C.SDLK_KP_CLEAR
	K_KP_CLEARENTRY        Keycode = C.SDLK_KP_CLEARENTRY
	K_KP_BINARY            Keycode = C.SDLK_KP_BINARY
	K_KP_OCTAL             Keycode = C.SDLK_KP_OCTAL
	K_KP_DECIMAL           Keycode = C.SDLK_KP_DECIMAL
	K_KP_HEXADECIMAL       Keycode = C.SDLK_KP_HEXADECIMAL
	K_LCTRL                Keycode = C.SDLK_LCTRL
	K_LSHIFT               Keycode = C.SDLK_LSHIFT
	K_LALT                 Keycode = C.SDLK_LALT
	K_LGUI                 Keycode = C.SDLK_LGUI
	K_RCTRL                Keycode = C.SDLK_RCTRL
	K_RSHIFT               Keycode = C.SDLK_RSHIFT
	K_RALT                 Keycode = C.SDLK_RALT
	K_RGUI                 Keycode = C.SDLK_RGUI
	K_MODE                 Keycode = C.SDLK_MODE
	K_SLEEP                Keycode = C.SDLK_SLEEP
	K_WAKE                 Keycode = C.SDLK_WAKE
	K_CHANNEL_INCREMENT    Keycode = C.SDLK_CHANNEL_INCREMENT
	K_CHANNEL_DECREMENT    Keycode = C.SDLK_CHANNEL_DECREMENT
	K_MEDIA_PLAY           Keycode = C.SDLK_MEDIA_PLAY
	K_MEDIA_PAUSE          Keycode = C.SDLK_MEDIA_PAUSE
	K_MEDIA_RECORD         Keycode = C.SDLK_MEDIA_RECORD
	K_MEDIA_FAST_FORWARD   Keycode = C.SDLK_MEDIA_FAST_FORWARD
	K_MEDIA_REWIND         Keycode = C.SDLK_MEDIA_REWIND
	K_MEDIA_NEXT_TRACK     Keycode = C.SDLK_MEDIA_NEXT_TRACK
	K_MEDIA_PREVIOUS_TRACK Keycode = C.SDLK_MEDIA_PREVIOUS_TRACK
	K_MEDIA_STOP           Keycode = C.SDLK_MEDIA_STOP
	K_MEDIA_EJECT          Keycode = C.SDLK_MEDIA_EJECT
	K_MEDIA_PLAY_PAUSE     Keycode = C.SDLK_MEDIA_PLAY_PAUSE
	K_MEDIA_SELECT         Keycode = C.SDLK_MEDIA_SELECT
	K_AC_NEW               Keycode = C.SDLK_AC_NEW
	K_AC_OPEN              Keycode = C.SDLK_AC_OPEN
	K_AC_CLOSE             Keycode = C.SDLK_AC_CLOSE
	K_AC_EXIT              Keycode = C.SDLK_AC_EXIT
	K_AC_SAVE              Keycode = C.SDLK_AC_SAVE
	K_AC_PRINT             Keycode = C.SDLK_AC_PRINT
	K_AC_PROPERTIES        Keycode = C.SDLK_AC_PROPERTIES
	K_AC_SEARCH            Keycode = C.SDLK_AC_SEARCH
	K_AC_HOME              Keycode = C.SDLK_AC_HOME
	K_AC_BACK              Keycode = C.SDLK_AC_BACK
	K_AC_FORWARD           Keycode = C.SDLK_AC_FORWARD
	K_AC_STOP              Keycode = C.SDLK_AC_STOP
	K_AC_REFRESH           Keycode = C.SDLK_AC_REFRESH
	K_AC_BOOKMARKS         Keycode = C.SDLK_AC_BOOKMARKS
	K_SOFTLEFT             Keycode = C.SDLK_SOFTLEFT
	K_SOFTRIGHT            Keycode = C.SDLK_SOFTRIGHT
	K_CALL                 Keycode = C.SDLK_CALL
	K_ENDCALL              Keycode = C.SDLK_ENDCALL
)

type Keymod uint16

// Scancode

type Scancode int32

// Mouse

type MouseID uint32

//go:generate stringer -type MouseButton -trimprefix BUTTON_

type MouseButton uint8

const (
	BUTTON_LEFT   MouseButton = C.SDL_BUTTON_LEFT
	BUTTON_MIDDLE MouseButton = C.SDL_BUTTON_MIDDLE
	BUTTON_RIGHT  MouseButton = C.SDL_BUTTON_RIGHT
	BUTTON_X1     MouseButton = C.SDL_BUTTON_X1
	BUTTON_X2     MouseButton = C.SDL_BUTTON_X2
)

type MouseButtonFlags uint32

func (window *Window) SetWindowRelativeMouseMode(enabled bool) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !C.SDL_SetWindowRelativeMouseMode((*C.SDL_Window)(window), C.bool(enabled)) {
		return getError()
	}
	return nil
}

// Joystick

type JoystickID uint32

// Gamepad

type Gamepad C.SDL_Gamepad

//go:generate stringer -type GamepadType -trimprefix GAMEPAD_TYPE_

type GamepadType C.SDL_GamepadType

const (
	GAMEPAD_TYPE_XBOX360 GamepadType = C.SDL_GAMEPAD_TYPE_XBOX360
	GAMEPAD_TYPE_XBOXONE GamepadType = C.SDL_GAMEPAD_TYPE_XBOXONE
	GAMEPAD_TYPE_PS3     GamepadType = C.SDL_GAMEPAD_TYPE_PS3
	GAMEPAD_TYPE_PS4     GamepadType = C.SDL_GAMEPAD_TYPE_PS4
	GAMEPAD_TYPE_PS5     GamepadType = C.SDL_GAMEPAD_TYPE_PS5
)

//go:generate stringer -type GamepadButton -trimprefix GAMEPAD_BUTTON_

type GamepadButton uint32

const (
	GAMEPAD_BUTTON_SOUTH          GamepadButton = C.SDL_GAMEPAD_BUTTON_SOUTH
	GAMEPAD_BUTTON_EAST           GamepadButton = C.SDL_GAMEPAD_BUTTON_EAST
	GAMEPAD_BUTTON_WEST           GamepadButton = C.SDL_GAMEPAD_BUTTON_WEST
	GAMEPAD_BUTTON_NORTH          GamepadButton = C.SDL_GAMEPAD_BUTTON_NORTH
	GAMEPAD_BUTTON_BACK           GamepadButton = C.SDL_GAMEPAD_BUTTON_BACK
	GAMEPAD_BUTTON_START          GamepadButton = C.SDL_GAMEPAD_BUTTON_START
	GAMEPAD_BUTTON_LEFT_STICK     GamepadButton = C.SDL_GAMEPAD_BUTTON_LEFT_STICK
	GAMEPAD_BUTTON_RIGHT_STICK    GamepadButton = C.SDL_GAMEPAD_BUTTON_RIGHT_STICK
	GAMEPAD_BUTTON_LEFT_SHOULDER  GamepadButton = C.SDL_GAMEPAD_BUTTON_LEFT_SHOULDER
	GAMEPAD_BUTTON_RIGHT_SHOULDER GamepadButton = C.SDL_GAMEPAD_BUTTON_RIGHT_SHOULDER
	GAMEPAD_BUTTON_DPAD_UP        GamepadButton = C.SDL_GAMEPAD_BUTTON_DPAD_UP
	GAMEPAD_BUTTON_DPAD_DOWN      GamepadButton = C.SDL_GAMEPAD_BUTTON_DPAD_DOWN
	GAMEPAD_BUTTON_DPAD_LEFT      GamepadButton = C.SDL_GAMEPAD_BUTTON_DPAD_LEFT
	GAMEPAD_BUTTON_DPAD_RIGHT     GamepadButton = C.SDL_GAMEPAD_BUTTON_DPAD_RIGHT
	GAMEPAD_BUTTON_RIGHT_PADDLE1  GamepadButton = C.SDL_GAMEPAD_BUTTON_RIGHT_PADDLE1
	GAMEPAD_BUTTON_LEFT_PADDLE1   GamepadButton = C.SDL_GAMEPAD_BUTTON_LEFT_PADDLE1
	GAMEPAD_BUTTON_RIGHT_PADDLE2  GamepadButton = C.SDL_GAMEPAD_BUTTON_RIGHT_PADDLE2
	GAMEPAD_BUTTON_LEFT_PADDLE2   GamepadButton = C.SDL_GAMEPAD_BUTTON_LEFT_PADDLE2
)

//go:generate stringer -type GamepadAxis -trimprefix GAMEPAD_AXIS_

type GamepadAxis uint8

const (
	GAMEPAD_AXIS_LEFTX         GamepadAxis = C.SDL_GAMEPAD_AXIS_LEFTX
	GAMEPAD_AXIS_LEFTY         GamepadAxis = C.SDL_GAMEPAD_AXIS_LEFTY
	GAMEPAD_AXIS_RIGHTX        GamepadAxis = C.SDL_GAMEPAD_AXIS_RIGHTX
	GAMEPAD_AXIS_RIGHTY        GamepadAxis = C.SDL_GAMEPAD_AXIS_RIGHTY
	GAMEPAD_AXIS_LEFT_TRIGGER  GamepadAxis = C.SDL_GAMEPAD_AXIS_LEFT_TRIGGER
	GAMEPAD_AXIS_RIGHT_TRIGGER GamepadAxis = C.SDL_GAMEPAD_AXIS_RIGHT_TRIGGER
)

func GetGamepads() []JoystickID {
	var gamepadCount C.int
	gamepads := (*JoystickID)(C.SDL_GetGamepads(&gamepadCount))
	defer C.SDL_free(unsafe.Pointer(gamepads))
	return append([]JoystickID(nil), unsafe.Slice(gamepads, int(gamepadCount))...)
}

func OpenGamepad(instanceID JoystickID) (*Gamepad, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	gamepad := (*Gamepad)(C.SDL_OpenGamepad(C.SDL_JoystickID(instanceID)))
	if gamepad == nil {
		return nil, getError()
	}
	return gamepad, nil
}

func (g *Gamepad) Name() string {
	return C.GoString(C.SDL_GetGamepadName((*C.SDL_Gamepad)(g)))
}

func (g *Gamepad) Type() GamepadType {
	return GamepadType(C.SDL_GetGamepadType((*C.SDL_Gamepad)(g)))
}

func (g *Gamepad) HasSensor(sensorType SensorType) bool {
	return bool(C.SDL_GamepadHasSensor((*C.SDL_Gamepad)(g), C.SDL_SensorType(sensorType)))
}

func (g *Gamepad) SetSensorEnabled(sensorType SensorType, enabled bool) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !C.SDL_SetGamepadSensorEnabled((*C.SDL_Gamepad)(g), C.SDL_SensorType(sensorType), C.bool(enabled)) {
		return getError()
	}
	return nil
}

func (g *Gamepad) SetLED(red, green, blue uint8) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !C.SDL_SetGamepadLED((*C.SDL_Gamepad)(g), C.Uint8(red), C.Uint8(green), C.Uint8(blue)) {
		return getError()
	}
	return nil
}

func (g *Gamepad) SendEffect(effect []byte) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !C.SDL_SendGamepadEffect((*C.SDL_Gamepad)(g), unsafe.Pointer(unsafe.SliceData(effect)), C.int(len(effect))) {
		return getError()
	}
	return nil
}

func (g *Gamepad) Close() {
	C.SDL_CloseGamepad((*C.SDL_Gamepad)(g))
}

// Sensor

//go:generate stringer -type SensorType -trimprefix SENSOR_

type SensorType C.SDL_SensorType

const (
	SENSOR_ACCEL SensorType = C.SDL_SENSOR_ACCEL
	SENSOR_GYRO  SensorType = C.SDL_SENSOR_GYRO
)

// Audio

//go:generate stringer -type AudioFormat -trimprefix AUDIO_

type AudioFormat uint16

const (
	AUDIO_F32 AudioFormat = C.SDL_AUDIO_F32
)

type AudioDeviceID C.SDL_AudioDeviceID

const (
	AUDIO_DEVICE_DEFAULT_PLAYBACK  AudioDeviceID = C.SDL_AUDIO_DEVICE_DEFAULT_PLAYBACK
	AUDIO_DEVICE_DEFAULT_RECORDING AudioDeviceID = C.SDL_AUDIO_DEVICE_DEFAULT_RECORDING
)

type AudioSpec struct {
	_          structs.HostLayout
	Format     AudioFormat
	Channels   int32
	SampleRate int32
}

func _() {
	var x [1]int
	_ = x[unsafe.Sizeof(AudioSpec{})-unsafe.Sizeof(C.SDL_AudioSpec{})]
}

type AudioStream C.SDL_AudioStream

func (device AudioDeviceID) Resume() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !C.SDL_ResumeAudioDevice(C.SDL_AudioDeviceID(device)) {
		return getError()
	}
	return nil
}

func OpenAudioDeviceStream(devId AudioDeviceID, spec *AudioSpec /*, callback func() */) (*AudioStream, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	stream := (*AudioStream)(C.SDL_OpenAudioDeviceStream(C.SDL_AudioDeviceID(devId), (*C.SDL_AudioSpec)(unsafe.Pointer(spec)), nil, nil))
	if stream == nil {
		return nil, getError()
	}
	return stream, nil
}

func (s *AudioStream) Device() AudioDeviceID {
	return AudioDeviceID(C.SDL_GetAudioStreamDevice((*C.SDL_AudioStream)(s)))
}

func (s *AudioStream) Queued() int {
	return int(C.SDL_GetAudioStreamQueued((*C.SDL_AudioStream)(s)))
}

func (s *AudioStream) Write(b []byte) (int, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if !C.SDL_PutAudioStreamData((*C.SDL_AudioStream)(s), unsafe.Pointer(unsafe.SliceData(b)), C.int(len(b))) {
		return 0, getError()
	}
	return len(b), nil
}

// Timer

func TicksNS() uint64 {
	return uint64(C.SDL_GetTicksNS())
}

// Internal helpers

// Must be called on the same OS thread that did the SDL call.
func getError() error {
	return errors.New(C.GoString(C.SDL_GetError()))
}

func cstring(s string) *C.char {
	a := make([]byte, len(s)+1)
	copy(a, s)
	return (*C.char)(unsafe.Pointer(unsafe.SliceData(a)))
}
