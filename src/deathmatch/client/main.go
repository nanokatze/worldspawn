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
	"path"
	"reflect"
	"runtime"
	"runtime/trace"
	"slices"
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
	"worldspawn/internal/ecs"
	sfx "worldspawn/internal/fuckwwise"
	"worldspawn/internal/fuckwwise/opusfile"
	"worldspawn/internal/fuckwwise/wav"
	"worldspawn/internal/pathtracer"
	"worldspawn/internal/pathtracer/matc"
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

var fn uint32

// Must be called with redrawMu held.
func redrawLocked() bool {
	defer trace.StartRegion(context.Background(), "Redraw (redrawMu held)").End()

	if swapchain == nil {
		return false
	}

	var jq gpu.JobQueue

	conf := config.Load()

	select {
	case update := <-clientRenderer.sceneUpdates:
		clientRenderer.scene.EnqueueUpdate(&jq, update.SceneUpdate, 0)
	default:
	}

	clientRenderer.stuffMu.Lock()
	t := 1.0
	if !conf.DontInterpolate {
		// TODO: we need to be able to lock sceneMu for this but we can't.
		// We should make our own scene type with the timestamps and stuff.
		t = min(max(float64(sdl.TicksNS()-clientRenderer.t0sdl)/float64(clientRenderer.t1sdl-clientRenderer.t0sdl), 0), 1)
	}
	ourCamera := clientRenderer.ourCamera
	clientRenderer.stuffMu.Unlock()

	// swapchainImage := swapchain.Image(swapchainImageIndex)

	swapchainImage.EnqueueInit(&jq)

	clientRenderer.scene.Render(
		&jq,
		pathtracer.Film{
			Extent: swapchainImage.Extent(),
			Color:  swapchainImage,
		},
		float32(t),
		fn,
		&ourCamera)
	fn++

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

// TODO: should be created on demand
// TODO: split into two parts
var clientRenderer = &idk{
	transformT0: make([]geometry.TRS3, 10000),

	sceneUpdates: make(chan *sceneUpdate, 1),

	scene: pathtracer.NewScene(10000, 5),

	sfxScene: &sfx.Scene{
		Instance: make([]sfx.Instance, 10000),
	},
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

// TODO: split into two
type idk struct {
	transformT0 []geometry.TRS3

	// Update being filled out. This may persist between multiple game updates
	// if sceneUpdates queue was full.
	stagingUpdate *sceneUpdate

	// TODO: this should be a part of sceneUpdate. We also need the "currently
	// being rendered time interval" so that we can map input back to that.
	stuffMu        sync.Mutex
	t0sdl, t1sdl   uint64 // TODO: special type to represent SDL ticks?
	t0game, t1game game.Time
	// TODO: what if we want to pass multiple cameras to the composition
	// pipeline?
	ourCamera pathtracer.Camera

	// Queue of updates, consumed by redrawLocked.
	sceneUpdates chan *sceneUpdate

	// Only used by the redrawLocked
	scene *pathtracer.Scene

	// Uhh
	sfxScene *sfx.Scene
}

func (sr *idk) Tick(w *game.Scene, playerID ecs.ID, t0, t1 game.Time, frameDuration time.Duration) {
	conf := config.Load()

	if sr.stagingUpdate == nil {
		// TODO: pool this stuff
		sr.stagingUpdate = &sceneUpdate{SceneUpdate: pathtracer.NewSceneDirty(10000)}
	}
	update := sr.stagingUpdate

	{
		for i := range update.Parent {
			update.Parent[i] = 0
		}

		for i := range update.Instance {
			update.Instance[i].Transform = 0
		}

		update.Sky = texture(w.Sky).Image

		playerEntity, _ := w.Entity.Load(playerID)
		fpsCharacter := playerEntity.(game.FPSCharacter)

		for id, tr := range w.TranslationRotation.All() {
			cosmeticOffset, _ := w.CosmeticOffset.Load(id)

			i := id.Index()

			parent, hasParent := w.Parent.Load(id)
			if hasParent {
				update.Parent[i] = parent.Index()
			}

			var offset geometry.Vec3
			if !conf.DisableCosmeticOffset {
				offset = cosmeticOffset.Eval(w.Now)
			}

			transformT0 := sr.transformT0[i]
			transformT1 := geometry.TRS3{
				Translation: tr.Translation.Add(geometry.DVec3FromVec3(offset)).Vec3(),
				Rotation:    tr.Rotation,
				Scale:       w.GetScale(id),
			}

			update.TransformT0[i] = transformT0
			update.TransformT1[i] = transformT1
			sr.transformT0[i] = transformT1
		}

		// TODO: we need to split operations on caches into probe and fetch, so
		// that when a probe fails, we spawn a new goroutine with fetch and
		// everything that follows.

		for id, v := range ecs.Join(w.RenderingGeometry, w.TranslationRotation) {
			entity, hasEntity := w.Entity.Load(id)
			renderingGeometry := v.V1

			i := id.Index()

			mask := uint8(0b11)

			viewmodel, hasViewmodel := w.Viewmodel2.Load(id)
			if hasViewmodel {
				switch viewmodel.Mode {
				case 1:
					mask = 0b01
				case 2:
					mask = 0b10
				}
			}

			mesh := getmesh(renderingGeometry)

			update.Mesh[i] = mesh.re

			// TODO: stop doing slices.Clone
			update.Materials[i] = slices.Clone(mesh.re.DefaultMaterials)

			if hasEntity {
				for j := range update.Materials[i] {
					m := &update.Materials[i][j]
					// TODO: do we call this params or args?
					args := reflect.NewAt(m.Material.ParamStruct, unsafe.Pointer(&m.Args)).Elem()
					// TODO: precompile Gather
					matc.GatherArgs(args, reflect.ValueOf(entity), m.Material.ParamNames)
				}
			}

			update.Instance[i].Mask = mask
			update.Instance[i].Transform = i
		}

		// TODO: this should not exist and be part of sceneUpdate
		sr.stuffMu.Lock()
		sr.t0sdl = sdl.TicksNS()
		sr.t1sdl = sr.t0sdl + uint64(frameDuration) // depends on timescale
		sr.t0game = t0
		sr.t1game = t1
		// TODO: factor out into a function, this gets reused in Subtick
		sr.ourCamera = pathtracer.Camera{
			Transform:     update.Transform(fpsCharacter.Camera.Index(), 0),
			FieldOfView:   float32(geometry.Radians(67.5)),
			NearClipPlane: 0.01,
		}
		sr.stuffMu.Unlock()
	}

	select {
	case sr.sceneUpdates <- update:
		sr.stagingUpdate = nil
	default:
	}

	// Ughhhhhhh
	{
		playerEntity, _ := w.Entity.Load(playerID)
		fpsCharacter := playerEntity.(game.FPSCharacter)
		camera := update.Transform(fpsCharacter.Camera.Index(), 0)
		cameraPos := geometry.Vec3{camera[0][3], camera[1][3], camera[2][3]}

		scene := sr.sfxScene

		a := int64(t1.Sub(t0) * 48000 / 1e9)

		clear(scene.Instance)

		for id, soundEffect := range w.SoundEffect.All() {
			positionRotation, _ := w.TranslationRotation.Load(id)
			scale, _ := w.Scale.Load(id)

			// TODO: take hierarchy into account
			xform := geometry.TRS3{
				Translation: positionRotation.Translation.Vec3(), // TODO: we should also be applying cosmetic offset like in video
				Rotation:    positionRotation.Rotation,
				Scale:       scale,
			}.Mat4x4()

			effect, ok := sources[soundEffect.Effect]
			if !ok {
				f, err := game.Data.Open(soundEffect.Effect)
				if err != nil {
					// TODO: should be non-fatal
					panic(fmt.Sprintf("failed to open file %v", soundEffect.Effect))
				}

				switch path.Ext(soundEffect.Effect) {
				case ".wav":
					reader, _ := wav.NewReader(f.(io.ReaderAt))
					samples, _ := readSamples(reader, reader.Format())
					effect = &sfx.Source{
						Samples: extractChannel(samples, reader.Channels(), 0),
					}

				case ".opus":
					reader, _ := opusfile.NewReader(f)
					samples, _ := readSamples(reader, sfx.FORMAT_F32)
					effect = &sfx.Source{
						Samples: extractChannel(samples, reader.Channels(), 0),
					}

				default:
					panic("unsupported")
				}

				sources[soundEffect.Effect] = effect
			}

			scene.Instance[id.Index()] = sfx.Instance{
				Transform: xform,
				Samples:   effect.Samples,
				PlayTime:  int64(soundEffect.PlayTime.Sub(game.Time(0)) * 48000 / 1e9),
			}
		}

		// TODO: let us do multiple audio renders per frame. Should be nice for
		// sessions with long ticks
		renderAudio(sr.sfxScene, cameraPos, int64(t0.Sub(game.Time(0))*48000/1e9), a)
	}
}

func (sr *idk) Subtick(w *game.Scene, playerID ecs.ID) {
	// re.stuffMu.Lock()
	// defer re.stuffMu.Unlock()

	// TODO: we'll need to fix camera shenanigans first

	// playerEntity, _ := w.Entity.Load(playerID)
	// fpsCharacter := playerEntity.(game.FPSCharacter)
	// cameraID := fpsCharacter.Camera
	// tr, _ := w.TranslationRotation.Load(cameraID)

	// rot := tr.Rotation

	// // TODO: chase the entire parent chain and update that as well?

	// clientRenderer.scene.TransformT0[cameraID.Index()].Rotation = rot
	// clientRenderer.scene.TransformT1[cameraID.Index()].Rotation = rot

	// clientRenderer.ourCamera = renderer.Camera{
	// 	Transform:     clientRenderer.scene.Transform(cameraID.Index(), 0),
	// 	FieldOfView:   float32(geometry.Radians(67.5)),
	// 	NearClipPlane: 0.01,
	// }
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
