package main

import (
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/geometry-go"
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
	"worldspawn/sdl"
)

// TODO: rename? though I guess this is the correct name for it
type mainWindow struct {
	sdlWindow *sdl.Window

	resizeCond sync.Cond
	redrawMu   sync.Mutex

	swapchain      *gpu.Swapchain
	swapchainImage *gpu.Image
}

func newMainWindow() *mainWindow {
	conf := config.Load()

	sdlWindow, err := sdl.CreateWindow(
		sdl.WithStringProperty(sdl.PROP_WINDOW_CREATE_TITLE_STRING, "Wo̅r̅l̅d̅s̅p̅a̅w̅n̅"),
		sdl.WithBooleanProperty(sdl.PROP_WINDOW_CREATE_VULKAN_BOOLEAN, true),
		sdl.WithBooleanProperty(sdl.PROP_WINDOW_CREATE_RESIZABLE_BOOLEAN, true),
		sdl.WithBooleanProperty(sdl.PROP_WINDOW_CREATE_HIGH_PIXEL_DENSITY_BOOLEAN, true),
		// TODO: minimum window size?
		sdl.WithNumberProperty(sdl.PROP_WINDOW_CREATE_WIDTH_NUMBER, int64(conf.Presentation.Resolution[0])),
		sdl.WithNumberProperty(sdl.PROP_WINDOW_CREATE_HEIGHT_NUMBER, int64(conf.Presentation.Resolution[1])),
	)
	if err != nil {
		panic(fmt.Sprintf("sdl.CreateWindow: %v", err))
	}

	w := new(mainWindow)
	w.sdlWindow = sdlWindow
	w.resizeCond.L = &w.redrawMu

	// TODO: start goroutine that redraws into this window?

	return w
}

func (w *mainWindow) resize(extent [3]int) {
	if extent[2] != 1 {
		panic("bad")
	}
	if extent[0] <= 0 || extent[1] <= 0 {
		return // minimized, do nothing
	}

	w.redrawMu.Lock()
	defer w.redrawMu.Unlock()

	slog.Info("resize", "extent", extent)

	w.swapchain = gpu.NewSwapchain(&gpu.SwapchainConfig{
		Window:     w.sdlWindow,
		ColorSpace: vk.COLOR_SPACE_SRGB_NONLINEAR_KHR,
		ImageConfig: &gpu.ImageConfig{
			Dim:       gpu.ImageDim2D,
			Extent:    extent,
			Layers:    1,
			MipLevels: 1,
			Samples:   1,
			Format:    vk.FORMAT_R8G8B8A8_SRGB,
			Usage:     gpu.ImageUsageAttachment,
		},
		OldSwapchain: w.swapchain,
	})

	gpu.NewPresentableImageTest(
		w.sdlWindow,
		&gpu.ImageConfig{
			Dim:       gpu.ImageDim2D,
			Extent:    extent,
			Layers:    1,
			MipLevels: 1,
			Samples:   1,
			Format:    vk.FORMAT_R8G8B8A8_SRGB,
			Usage:     gpu.ImageUsageAttachment,
		})

	w.swapchainImage = gpu.NewImage(&gpu.ImageConfig{
		Dim:       gpu.ImageDim2D,
		Extent:    extent,
		Layers:    1,
		MipLevels: 1,
		Samples:   1,
		Format:    vk.FORMAT_R8G8B8A8_UNORM,
		Usage:     gpu.ImageUsageLoadStore | gpu.ImageUsageAttachment,
	})

	// Redraw a single frame at this size.
	w.redrawLocked()

	w.resizeCond.Broadcast()
}

func (w *mainWindow) redraw() {
	w.redrawMu.Lock()
	defer w.redrawMu.Unlock()

	if !w.redrawLocked() {
		// Present failed, wait for resize.
		w.resizeCond.Wait()
	}
}

// TODO: factor window-specific code into a type (mainWindow or gameWindow or
// whatever)

// Must be called with redrawMu held.
func (w *mainWindow) redrawLocked() bool {
	if w.swapchain == nil {
		return false
	}

	var jq gpu.JobQueue

	w.swapchainImage.EnqueueInit(&jq)

	gameRenderer.Render(&jq, w.swapchainImage)

	// TODO: it would probably be a good idea to inject overlay rendering into
	// Render so that we can avoid breaking the render pass.

	presentationOk := w.swapchain.Present2(&jq, w.swapchainImage)

	// TODO: frames-in-flight

	jq.WaitForIdle()

	return presentationOk
}

type flickStick struct {
	activated      bool
	deflection     geometry.Vec2
	lastDeflection geometry.Vec2
}

var flickStickTest flickStick

func sdlTimeToGameTime(ticks uint64) game.Time {
	gameRenderer.stuffMu.Lock()
	defer gameRenderer.stuffMu.Unlock()

	// If we have DontInterpolate set, we'll want t = 1
	t := min(max(float64(ticks-gameRenderer.t0sdl)/float64(gameRenderer.t1sdl-gameRenderer.t0sdl), 0), 1)

	return gameRenderer.t0game.Add(time.Duration(float64(gameRenderer.t1game-gameRenderer.t0game) * t))
}

// GAMEPAD_BUTTON_START and K_ESC act as ways to switch between the menu and the
// game
//
// GAMEPAD_BUTTON_BACK acts as Tab in-game (i.e. shows scoreboard or w/e) but
// nothing otherwise

// TODO: we can filter out unchanging actions here
func (w *mainWindow) handleInput(e any) {
	var cmds []game.TimestampedInputCmd

	conf := config.Load()

	switch e := e.(type) {
	case *sdl.WindowPixelSizeChangedEvent:
		w.resize([3]int{int(e.Data1), int(e.Data2), 1})

	case *sdl.KeyDownEvent:
		etime := sdlTimeToGameTime(e.Timestamp)

		if action, ok := conf.Controls.KeyActions[e.Key]; ok {
			cmds = game.AppendAction(cmds, etime, action, 1)
		}

	case *sdl.KeyUpEvent:
		etime := sdlTimeToGameTime(e.Timestamp)

		if action, ok := conf.Controls.KeyActions[e.Key]; ok {
			cmds = game.AppendAction(cmds, etime, action, 0)
		}

	case *sdl.MouseMotionEvent:
		etime := sdlTimeToGameTime(e.Timestamp)

		cmds = game.AppendAction(cmds, etime, game.ActionDLookX, e.XRel*0.0005)
		cmds = game.AppendAction(cmds, etime, game.ActionDLookY, e.YRel*0.0005)

	case *sdl.MouseButtonDownEvent:
		etime := sdlTimeToGameTime(e.Timestamp)

		if action, ok := conf.Controls.MouseButtonActions[sdl.MouseButton(e.Button)]; ok {
			cmds = game.AppendAction(cmds, etime, action, 1)
		}

	case *sdl.MouseButtonUpEvent:
		etime := sdlTimeToGameTime(e.Timestamp)

		if action, ok := conf.Controls.MouseButtonActions[sdl.MouseButton(e.Button)]; ok {
			cmds = game.AppendAction(cmds, etime, action, 0)
		}

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

		if action, ok := conf.Controls.GamepadButtonActions[sdl.GamepadButton(e.Button)]; ok {
			cmds = game.AppendAction(cmds, etime, action, 1)
		}

	case *sdl.GamepadButtonUpEvent:
		etime := sdlTimeToGameTime(e.Timestamp)

		if action, ok := conf.Controls.GamepadButtonActions[sdl.GamepadButton(e.Button)]; ok {
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
