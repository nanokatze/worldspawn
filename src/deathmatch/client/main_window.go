package main

import (
	"flag"
	"log"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/geometry-go"
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
	sfx "worldspawn/internal/fuckwwise"
	"worldspawn/internal/pathtracer"
	"worldspawn/sdl"
	"worldspawn/sdlapp"
)

// TODO: see if we can kill this and fold things back into main()
// TODO: I'd probably prefer if it wasn't called "main" but something that
// better describes its purpose (like a file explorer window would be
// fileExplorerWindow or whatever.)
type mainWindow struct {
	sdlWindow *sdl.Window // TODO: kill?

	session atomic.Pointer[Client]

	flickStickTest flickStick

	resized        chan struct{}
	redrawMu       sync.Mutex
	redrawJQ       gpu.JobQueue
	swapchain      *gpu.Swapchain
	swapchainImage *gpu.Image
	renderer       *renderer // TODO: this could be an interface probably
}

func (w *mainWindow) Run() {
	conf := config.Load()

	// TODO: delegate creating the window to the user? If we make windows work
	// like gioui windows, we could let the user set the size and we'd set the
	// title.
	// TODO: it would also maybe be nice to tie windows with their swapchains.
	sdlWindow := sdlapp.CreateWindow(
		sdl.WithBooleanProperty(sdl.PROP_WINDOW_CREATE_VULKAN_BOOLEAN, true),
		sdl.WithBooleanProperty(sdl.PROP_WINDOW_CREATE_HIGH_PIXEL_DENSITY_BOOLEAN, true),
		sdl.WithStringProperty(sdl.PROP_WINDOW_CREATE_TITLE_STRING, "Wo̅r̅l̅d̅s̅p̅a̅w̅n̅"),
		// TODO: minimum window size as a safeguard?
		sdl.WithBooleanProperty(sdl.PROP_WINDOW_CREATE_RESIZABLE_BOOLEAN, true),
		sdl.WithNumberProperty(sdl.PROP_WINDOW_CREATE_WIDTH_NUMBER, int64(conf.Presentation.Resolution[0])),
		sdl.WithNumberProperty(sdl.PROP_WINDOW_CREATE_HEIGHT_NUMBER, int64(conf.Presentation.Resolution[1])),
	)

	w.sdlWindow = sdlWindow

	w.resized = make(chan struct{}, 1)

	// TODO: the renderer should autoresize its resources on Ticks
	w.renderer = &renderer{
		lastTransform: make([]geometry.TRS3, 10000),

		updates: make(chan *sceneUpdate, 1),

		scene: pathtracer.NewScene(10000, 5),

		sfxScene: &sfx.Scene{
			Instance: make([]sfx.Instance, 10000),
		},
	}

	if err := sdlWindow.SetWindowRelativeMouseMode(true); err != nil {
		slog.Warn("failed to set relative mouse mode", "err", err)
	}

	raddr := flag.Arg(0)

	session, err := newClient(w.renderer, raddr)
	if err != nil {
		log.Fatal(err)
	}

	w.session.Store(session)

	go func() {
		for {
			<-w.resized
			for {
				if !w.redraw() {
					break
				}
			}
		}
	}()

	for {
		event := <-sdlapp.Events(sdlWindow)

		conf := config.Load()

		var cmds []game.TimestampedInputCmd
		switch e := event.(type) {
		case *sdl.WindowPixelSizeChangedEvent:
			w.resize([2]int{int(e.Data1), int(e.Data2)})

		case *sdl.KeyDownEvent:
			etime := w.sdlTimeToGameTime(e.Timestamp)

			if action, ok := conf.Controls.KeyActions[e.Key]; ok {
				cmds = game.AppendAction(cmds, etime, action, 1)
			}

		case *sdl.KeyUpEvent:
			etime := w.sdlTimeToGameTime(e.Timestamp)

			if action, ok := conf.Controls.KeyActions[e.Key]; ok {
				cmds = game.AppendAction(cmds, etime, action, 0)
			}

		case *sdl.MouseMotionEvent:
			etime := w.sdlTimeToGameTime(e.Timestamp)

			cmds = game.AppendAction(cmds, etime, game.ActionDLookX, e.XRel*0.0005)
			cmds = game.AppendAction(cmds, etime, game.ActionDLookY, e.YRel*0.0005)

		case *sdl.MouseButtonDownEvent:
			etime := w.sdlTimeToGameTime(e.Timestamp)

			if action, ok := conf.Controls.MouseButtonActions[sdl.MouseButton(e.Button)]; ok {
				cmds = game.AppendAction(cmds, etime, action, 1)
			}

		case *sdl.MouseButtonUpEvent:
			etime := w.sdlTimeToGameTime(e.Timestamp)

			if action, ok := conf.Controls.MouseButtonActions[sdl.MouseButton(e.Button)]; ok {
				cmds = game.AppendAction(cmds, etime, action, 0)
			}

		case *sdl.GamepadAxisMotionEvent:
			etime := w.sdlTimeToGameTime(e.Timestamp)

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
				w.flickStickTest.deflection.X = value

			case sdl.GAMEPAD_AXIS_RIGHTY:
				w.flickStickTest.deflection.Y = value

			case sdl.GAMEPAD_AXIS_RIGHT_TRIGGER:
				cmds = game.AppendAction(cmds, etime, game.ActionAttack, step(0.9, value))
			}

		case *sdl.GamepadButtonDownEvent:
			etime := w.sdlTimeToGameTime(e.Timestamp)

			if sdl.GamepadButton(e.Button) == sdl.GAMEPAD_BUTTON_START {
				return
			}

			if action, ok := conf.Controls.GamepadButtonActions[sdl.GamepadButton(e.Button)]; ok {
				cmds = game.AppendAction(cmds, etime, action, 1)
			}

		case *sdl.GamepadButtonUpEvent:
			etime := w.sdlTimeToGameTime(e.Timestamp)

			if action, ok := conf.Controls.GamepadButtonActions[sdl.GamepadButton(e.Button)]; ok {
				cmds = game.AppendAction(cmds, etime, action, 0)
			}

		case *sdl.GamepadUpdateCompleteEvent:
			etime := w.sdlTimeToGameTime(e.Timestamp)

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

			activation := w.flickStickTest.deflection.Length() > 0.5

			if activation && !w.flickStickTest.activated {
				w.flickStickTest.lastDeflection = geometry.Vec2{X: 0, Y: -1}
			}
			w.flickStickTest.activated = activation

			if w.flickStickTest.activated {
				A := complex(w.flickStickTest.lastDeflection.X, w.flickStickTest.lastDeflection.Y)
				B := complex(w.flickStickTest.deflection.X, w.flickStickTest.deflection.Y)

				// We can normalize B and A and then we can just use B * conj(A)
				D := B / A

				dlookx := float32(math.Atan2(float64(imag(D)), float64(real(D))) / (2 * math.Pi))

				cmds = game.AppendAction(cmds, etime, game.ActionDLookX, dlookx)

				w.flickStickTest.lastDeflection = w.flickStickTest.deflection
			}
		}

		if len(cmds) > 0 {
			session := w.session.Load()
			session.HandleInput(cmds)
		}
	}
}

func (w *mainWindow) resize(size [2]int) {
	if size[0] <= 0 || size[1] <= 0 {
		return // minimized, do nothing
	}

	w.redrawMu.Lock()
	defer w.redrawMu.Unlock()

	slog.Info("resize", "size", size)

	w.swapchain = gpu.NewSwapchain(&gpu.SwapchainConfig{
		Window:     w.sdlWindow,
		ColorSpace: vk.COLOR_SPACE_SRGB_NONLINEAR_KHR,
		ImageConfig: &gpu.ImageConfig{
			Dim:       gpu.ImageDim2D,
			Extent:    [3]int{size[0], size[1], 1},
			Layers:    1,
			MipLevels: 1,
			Samples:   1,
			Format:    vk.FORMAT_R8G8B8A8_SRGB,
			Usage:     gpu.ImageUsageAttachment,
		},
		OldSwapchain: w.swapchain,
	})

	w.swapchainImage = gpu.NewImage(&gpu.ImageConfig{
		Dim:       gpu.ImageDim2D,
		Extent:    [3]int{size[0], size[1], 1},
		Layers:    1,
		MipLevels: 1,
		Samples:   1,
		Format:    vk.FORMAT_R8G8B8A8_UNORM,
		Usage:     gpu.ImageUsageLoadStore | gpu.ImageUsageAttachment,
	})

	// Redraw a single frame at this size.
	w.redrawLocked()

	select {
	case w.resized <- struct{}{}:
	default:
	}
}

func (w *mainWindow) redraw() bool {
	w.redrawMu.Lock()
	defer w.redrawMu.Unlock()

	return w.redrawLocked()
}

// Must be called with redrawMu held.
func (w *mainWindow) redrawLocked() bool {
	jq := &w.redrawJQ

	w.swapchainImage.EnqueueInit(jq)

	w.renderer.Render(jq, sdl.TicksNS(), w.swapchainImage)

	// TODO: it would probably be a good idea to inject overlay rendering into
	// Render so that we can avoid breaking the render pass. This should become
	// doable with the compositor.

	presentationOk := w.swapchain.Present2(jq, w.swapchainImage)

	// TODO: frames-in-flight
	jq.WaitForIdle()

	return presentationOk
}

type flickStick struct {
	activated      bool
	deflection     geometry.Vec2
	lastDeflection geometry.Vec2
}

func (w *mainWindow) sdlTimeToGameTime(ticks uint64) game.Time {
	w.renderer.stuffMu.Lock()
	defer w.renderer.stuffMu.Unlock()

	// If we have DontInterpolate set, we'll want t = 1
	t := min(max(float64(ticks-w.renderer.tm.t0sdl)/float64(w.renderer.tm.t1sdl-w.renderer.tm.t0sdl), 0), 1)

	return w.renderer.tm.t0game.Add(time.Duration(float64(w.renderer.tm.t1game-w.renderer.tm.t0game) * t))
}

// GAMEPAD_BUTTON_START and K_ESC act as ways to switch between the menu and the
// game
//
// GAMEPAD_BUTTON_BACK acts as Tab in-game (i.e. shows scoreboard or w/e) but
// nothing otherwise

func step(edge, x float32) float32 {
	if x < edge {
		return 0
	} else {
		return 1
	}
}
