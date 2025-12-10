package main

// #cgo LDFLAGS: -lphysics -lJolt
// #cgo LDFLAGS: -lm -lstdc++
import "C"

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math"
	"os"
	"runtime"
	"runtime/trace"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/geometry-go"
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
	sfx "worldspawn/internal/fuckwwise"
	"worldspawn/sdl"
)

var dataDir = flag.String("data", "data/cooked", "a")

// TODO: should this be in worldspawn
var messagePrinter = message.NewPrinter(language.English)

var window *sdl.Window

var redrawMu sync.Mutex
var resizeCond = sync.Cond{L: &redrawMu}

var swapchain *gpu.Swapchain
var swapchainImage *gpu.Image

var currentSession atomic.Pointer[Client]

var gamepad *sdl.Gamepad

func createWindow(props2 ...func(props sdl.PropertiesID) error) (*sdl.Window, error) {
	props, err := sdl.CreateProperties()
	if err != nil {
		return nil, err
	}
	defer props.Destroy()

	for _, f := range props2 {
		if err := f(props); err != nil {
			return nil, err
		}
	}

	return sdl.CreateWindowWithProperties(props)
}

// TODO: put sdl inits behind sync.Onces?

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	flag.Parse()

	log.SetFlags(0)

	config.Store(&Config{})

	// We don't use SDL event watcher to handle resizes as our redraw is too
	// slow to provide responsive size changes.
	//
	// For handling input, there appears to be marginal to no benefit over using
	// WaitEvents.

	initAudio()

	initGamepad()

	if err := sdl.InitSubSystem(sdl.INIT_VIDEO); err != nil {
		panic(fmt.Sprintf("failed to initialize SDL video subsystem: %v", err))
	}

	{
		var err error
		// TODO: make `window` not global, for the sake of tidier code
		window, err = createWindow(
			sdl.WithStringProperty(sdl.PROP_WINDOW_CREATE_TITLE_STRING, "Wo̅r̅l̅d̅s̅p̅a̅w̅n̅"),
			// sdl.WithBooleanProperty(sdl.PROP_WINDOW_CREATE_WAYLAND_SURFACE_ROLE_CUSTOM_BOOLEAN, true),
			sdl.WithBooleanProperty(sdl.PROP_WINDOW_CREATE_VULKAN_BOOLEAN, true),
			sdl.WithBooleanProperty(sdl.PROP_WINDOW_CREATE_RESIZABLE_BOOLEAN, true),
			sdl.WithBooleanProperty(sdl.PROP_WINDOW_CREATE_HIGH_PIXEL_DENSITY_BOOLEAN, true),
			sdl.WithNumberProperty(sdl.PROP_WINDOW_CREATE_WIDTH_NUMBER, int64(1280)),
			sdl.WithNumberProperty(sdl.PROP_WINDOW_CREATE_HEIGHT_NUMBER, int64(800)))
		if err != nil {
			log.Fatal(err)
		}
	}

	go func() {
		for {
			redraw()
		}
	}()

	if err := window.SetWindowRelativeMouseMode(true); err != nil {
		slog.Warn("failed to set relative mouse mode", "err", err)
	}

	slog.Info("gamepads", "gamepads", sdl.GetGamepads())

	// TODO: open all gamepads we have here
	gamepad, _ = sdl.OpenGamepad(sdl.GetGamepads()[0])

	slog.Info("gamepad", "gamepad", gamepad)

	raddr := flag.Arg(0)

	// TODO: should newRemoteSession do the logging instead? Yes.

	game.Data = os.DirFS(*dataDir)

	session, err := newClient(clientRenderer, raddr)
	if err != nil {
		log.Fatal(err)
	}

	currentSession.Store(session)

eventLoop:
	for {
		e, err := sdl.WaitEvent()
		if err != nil {
			log.Fatalln("WaitEvent failed", err)
		}

		switch e := e.(type) {
		case *sdl.QuitEvent:
			break eventLoop

		default:
			handleInput(e)
		}
	}
}

func resize(width, height int) {
	if width <= 0 || height <= 0 {
		return // minimized, do nothing
	}

	redrawMu.Lock()
	defer redrawMu.Unlock()

	slog.Info("resize", "width", width, "height", height)

	currentExtent := [3]int{width, height, 1}

	swapchain = gpu.NewSwapchain(&gpu.SwapchainConfig{
		Window:     window,
		ColorSpace: vk.COLOR_SPACE_SRGB_NONLINEAR_KHR,
		ImageConfig: &gpu.ImageConfig{
			Dim:       gpu.ImageDim2D,
			Extent:    currentExtent,
			Layers:    1,
			MipLevels: 1,
			Samples:   1,
			Format:    vk.FORMAT_R8G8B8A8_SRGB,
			Usage:     gpu.ImageUsageAttachment,
		},
		OldSwapchain: swapchain,
	})

	gpu.NewPresentableImageTest(
		window,
		&gpu.ImageConfig{
			Dim:       gpu.ImageDim2D,
			Extent:    currentExtent,
			Layers:    1,
			MipLevels: 1,
			Samples:   1,
			Format:    vk.FORMAT_R8G8B8A8_SRGB,
			Usage:     gpu.ImageUsageAttachment,
		})

	swapchainImage = gpu.NewImage(&gpu.ImageConfig{
		Dim:       gpu.ImageDim2D,
		Extent:    currentExtent,
		Layers:    1,
		MipLevels: 1,
		Samples:   1,
		Format:    vk.FORMAT_R8G8B8A8_UNORM,
		Usage:     gpu.ImageUsageLoadStore | gpu.ImageUsageAttachment,
	})

	// Redraw a single frame at this size.
	redrawLocked()

	resizeCond.Broadcast()
}

func redraw() {
	defer trace.StartRegion(context.Background(), "Redraw").End()

	redrawMu.Lock()
	defer redrawMu.Unlock()

	if !redrawLocked() {
		// Present failed, wait for resize.
		resizeCond.Wait()
	}
}

// Must be called with redrawMu held.
func redrawLocked() bool {
	defer trace.StartRegion(context.Background(), "Redraw (redrawMu held)").End()

	if swapchain == nil {
		return false
	}

	var jq gpu.JobQueue

	swapchainImage.EnqueueInit(&jq)

	clientRenderer.Render(&jq)

	// TODO: it would probably be a good idea to inject overlay rendering into
	// Render so that we can avoid breaking the render pass.

	// Menu

	func() {
		if true {
			return
		}

		rp := gpu.BeginRendering(&jq,
			&gpu.RenderingConfig{
				ColorAttachments: []gpu.Attachment{
					{
						Image: swapchainImage.SubImage(
							swapchainImage.Dim(),
							vk.FORMAT_R8G8B8A8_SRGB,
							0, 1,
							0, 1),
						LoadOp:  vk.ATTACHMENT_LOAD_OP_LOAD,
						StoreOp: vk.ATTACHMENT_STORE_OP_STORE,
					},
				},
				RenderArea: vk.Rect2D{Extent: vk.Extent2D{Width: uint32(swapchainImage.Extent()[0]), Height: uint32(swapchainImage.Extent()[1])}},
				LayerCount: 1,
			})
		defer rp.End()

		rp.SetPrimitiveTopology(vk.PRIMITIVE_TOPOLOGY_TRIANGLE_LIST)
		rp.SetPrimitiveRestartEnable(false)

		rp.SetViewports([]vk.Viewport{
			{
				X:        0,
				Y:        0,
				Width:    float32(swapchainImage.Extent()[0]),
				Height:   float32(swapchainImage.Extent()[1]),
				MinDepth: 0,
				MaxDepth: 1,
			},
		})
		rp.SetScissors([]vk.Rect2D{
			{Extent: vk.Extent2D{Width: uint32(swapchainImage.Extent()[0]), Height: uint32(swapchainImage.Extent()[1])}},
		})

		rp.SetRasterizerDiscardEnable(false)
		rp.SetPolygonMode(vk.POLYGON_MODE_FILL)
		rp.SetCullMode(vk.CullModeFlags(vk.CULL_MODE_NONE))
		rp.SetFrontFace(vk.FRONT_FACE_COUNTER_CLOCKWISE)
		rp.SetDepthBiasEnable(false)

		rp.SetRasterizationSamples(1)
		rp.SetSampleMask(0b1)
		rp.SetAlphaToCoverageEnable(false)

		rp.SetDepthTestEnable(false)
		rp.SetStencilTestEnable(false)

		rp.SetColorBlendEnable(0, true)
		rp.SetColorBlendEquation(0,
			vk.ColorBlendEquationEXT{
				SrcColorBlendFactor: vk.BLEND_FACTOR_SRC_ALPHA,
				DstColorBlendFactor: vk.BLEND_FACTOR_ONE,
				ColorBlendOp:        vk.BLEND_OP_ADD,
				SrcAlphaBlendFactor: vk.BLEND_FACTOR_ZERO,
				DstAlphaBlendFactor: vk.BLEND_FACTOR_ONE,
				AlphaBlendOp:        vk.BLEND_OP_ADD,
			})
		rp.SetColorWriteMask(0, 0b1111)

		rp.SetShader(vk.SHADER_STAGE_VERTEX_BIT, nil)
		rp.SetShader(vk.SHADER_STAGE_FRAGMENT_BIT, nil)
	}()

	var presentationOk bool
	trace.WithRegion(context.Background(), "Presentation", func() {
		presentationOk = swapchain.Present2(&jq, swapchainImage)
	})

	// TODO: frames-in-flight

	jq.WaitForIdle()

	return presentationOk
}

func readSamples(r io.Reader, format sfx.Format) ([]float32, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	switch format {
	case sfx.FORMAT_S16:
		bufSNORM16 := unsafe.Slice((*int16)(unsafe.Pointer(unsafe.SliceData(buf))), len(buf)/2)
		bufFLOAT32 := make([]float32, len(bufSNORM16))
		for i := range bufFLOAT32 {
			bufFLOAT32[i] = max(float32(bufSNORM16[i])/32767.0, -1)
		}
		return bufFLOAT32, nil

	case sfx.FORMAT_F32:
		bufFLOAT32 := unsafe.Slice((*float32)(unsafe.Pointer(unsafe.SliceData(buf))), len(buf)/4)
		return bufFLOAT32, nil

	default:
		panic("unsupported format")
	}
}

func extractChannel(s []float32, channels, channel int) []float32 {
	if channels == 1 {
		return s
	}

	s2 := make([]float32, len(s)/channels)
	for i := range s2 {
		s2[i] = s[i*channels+channel]
	}
	return s2
}

type flickStick struct {
	activated      bool
	deflection     geometry.Vec2
	lastDeflection geometry.Vec2
}

var flickStickTest flickStick

func sdlTimeToGameTime(ticks uint64) game.Time {
	clientRenderer.stuffMu.Lock()
	defer clientRenderer.stuffMu.Unlock()

	// If we have DontInterpolate set, we'll want t = 1
	t := min(max(float64(ticks-clientRenderer.t0sdl)/float64(clientRenderer.t1sdl-clientRenderer.t0sdl), 0), 1)

	return clientRenderer.t0game.Add(time.Duration(float64(clientRenderer.t1game-clientRenderer.t0game) * t))
}

// https://github.com/libsdl-org/SDL/issues/4464 🥺

var keyActions = map[sdl.Keycode]int{
	sdl.K_SPACE: game.ActionJump,
	sdl.K_LCTRL: game.ActionCrouch,
}

var gamepadButtonActions = map[sdl.GamepadButton]int{
	sdl.GAMEPAD_BUTTON_DPAD_UP:    game.ActionSlot1,
	sdl.GAMEPAD_BUTTON_DPAD_DOWN:  game.ActionSlot3,
	sdl.GAMEPAD_BUTTON_DPAD_LEFT:  game.ActionSlot0,
	sdl.GAMEPAD_BUTTON_DPAD_RIGHT: game.ActionSlot2,
}

// GAMEPAD_BUTTON_START and K_ESC act as ways to switch between the menu and the
// game
//
// GAMEPAD_BUTTON_BACK acts as Tab in-game (i.e. shows scoreboard or w/e) but
// nothing otherwise

// TODO: we can filter out unchanging actions here

func handleInput(e any) {
	var cmds []game.TimestampedInputCmd

	switch e := e.(type) {
	case *sdl.WindowPixelSizeChangedEvent:
		resize(int(e.Data1), int(e.Data2))

	case *sdl.KeyDownEvent:
		etime := sdlTimeToGameTime(e.Timestamp)

		if action, ok := keyActions[e.Key]; ok {
			cmds = game.AppendAction(cmds, etime, action, 1)
		}

	case *sdl.KeyUpEvent:
		etime := sdlTimeToGameTime(e.Timestamp)

		if action, ok := keyActions[e.Key]; ok {
			cmds = game.AppendAction(cmds, etime, action, 0)
		}

	case *sdl.MouseMotionEvent:
		etime := sdlTimeToGameTime(e.Timestamp)

		cmds = game.AppendAction(cmds, etime, game.ActionDLookX, e.XRel*0.0005)
		cmds = game.AppendAction(cmds, etime, game.ActionDLookY, e.YRel*0.0005)

	case *sdl.GamepadAxisMotionEvent:
		etime := sdlTimeToGameTime(e.Timestamp)

		value := max(float32(e.Value)/32767, -1)

		switch e.Axis {
		case sdl.GAMEPAD_AXIS_LEFTX:
			if math.Abs(float64(value)) < 0.2 {
				value = 0
			}

			cmds = game.AppendAction(cmds, etime, game.ActionSetMovementVelocityX, value)

		case sdl.GAMEPAD_AXIS_LEFTY:
			if math.Abs(float64(value)) < 0.2 {
				value = 0
			}

			cmds = game.AppendAction(cmds, etime, game.ActionSetMovementVelocityY, -value)

		case sdl.GAMEPAD_AXIS_RIGHTX:
			flickStickTest.deflection.X = value

		case sdl.GAMEPAD_AXIS_RIGHTY:
			flickStickTest.deflection.Y = value

		case sdl.GAMEPAD_AXIS_RIGHT_TRIGGER:
			cmds = game.AppendAction(cmds, etime, game.ActionAttack, step(0.9, value))
		}

	case *sdl.GamepadButtonDownEvent:
		etime := sdlTimeToGameTime(e.Timestamp)

		if sdl.GamepadButton(e.Button) == sdl.GAMEPAD_BUTTON_START {
			return
		}

		if action, ok := gamepadButtonActions[sdl.GamepadButton(e.Button)]; ok {
			cmds = game.AppendAction(cmds, etime, action, 1)
		}

	case *sdl.GamepadButtonUpEvent:
		etime := sdlTimeToGameTime(e.Timestamp)

		if action, ok := gamepadButtonActions[sdl.GamepadButton(e.Button)]; ok {
			cmds = game.AppendAction(cmds, etime, action, 0)
		}

	case *sdl.GamepadUpdateCompleteEvent:
		etime := sdlTimeToGameTime(e.Timestamp)

		// TODO: batch gamepad-generated commands and only send them in response
		// to this event, rather than immediately.
		//
		// TODO: will Timestamp on this event be different from timestamps on
		// other gamepad events arriving before this? If so, we'll need to
		// adjust how we sent "atomic" batch of commands. Actually we want that
		// one way or another anyway...

		// Flick stick camera hacked in
		//
		// TODO: smooth out the initial flick
		// TODO: do not send inputs when the stick is being released

		activation := flickStickTest.deflection.Length() > 0.5

		if activation && !flickStickTest.activated {
			flickStickTest.lastDeflection = geometry.Vec2{X: 0, Y: -1}
		}
		flickStickTest.activated = activation

		if flickStickTest.activated {
			A := complex(flickStickTest.lastDeflection.X, flickStickTest.lastDeflection.Y)
			B := complex(flickStickTest.deflection.X, flickStickTest.deflection.Y)

			// We can normalize B and A and then we can just use B * conj(A)
			D := B / A

			dlookx := float32(math.Atan2(float64(imag(D)), float64(real(D))) / (2 * math.Pi))

			cmds = game.AppendAction(cmds, etime, game.ActionDLookX, dlookx)

			flickStickTest.lastDeflection = flickStickTest.deflection
		}
	}

	if len(cmds) > 0 {
		session := currentSession.Load()
		session.HandleInput(cmds)
	}
}

func step(edge, x float32) float32 {
	if x < edge {
		return 0
	} else {
		return 1
	}
}

func mustReadFile(filename string) []byte {
	data, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return data
}
