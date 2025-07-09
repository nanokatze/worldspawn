package main

// #cgo LDFLAGS: -lphysics -lJolt
// #cgo LDFLAGS: -lm -lstdc++
import "C"

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math"
	"os"
	"path"
	"runtime"
	"runtime/trace"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"worldspawn"
	"worldspawn/ecs"
	sfx "worldspawn/fuckwwise"
	"worldspawn/fuckwwise/opusfile"
	"worldspawn/fuckwwise/wav"
	"worldspawn/geometry-go"
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
	"worldspawn/renderer"
	"worldspawn/sdl"
)

// TODO: make a -data flag

// TODO: should be moved somewhere.
//
// TODO: It's also not clear whether the same printer should be used by both
// menu and the game, especially given that games might provide their own
// strings
var messagePrinter = message.NewPrinter(language.English)

var window *sdl.Window

var redrawMu sync.Mutex
var resizeCond = sync.Cond{L: &redrawMu}

var currentSession atomic.Pointer[Client]

var gamepad *sdl.Gamepad

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	flag.Parse()

	log.SetFlags(0)

	config.Store(&Config{})

	go func() {
		stdin := bufio.NewScanner(os.Stdin)
		stderr := os.Stderr

		for stdin.Scan() {
			cmd := strings.Fields(stdin.Text())
			if len(cmd) == 0 {
				continue
			}

			// TODO: use powershell/bash syntax. Config variables would be set
			// using VariableName=Value and commands would be run with Command
			// ...

			switch cmd[0] {
			case "Slot":
				/*
					slot, _ := strconv.Atoi(cmd[1])
					icmd := inputCommand2()
					icmd.Commands = append(icmd.Commands, worldspawn.Slot(slot))
					currentSession.Load().Input(icmd)
				*/
				panic("not implemented")

			case "DontInterpolate":
				updateConfig(func(conf *Config) { conf.DontInterpolate = true })

			case "DoInterpolate":
				updateConfig(func(conf *Config) { conf.DontInterpolate = false })

			default:
				fmt.Fprintf(stderr, "unknown command %s\n", cmd[0])
			}
		}
	}()

	if err := sdl.InitSubSystem(sdl.INIT_VIDEO); err != nil {
		panic(fmt.Sprintf("failed to initialize SDL video subsystem: %v", err))
	}

	// We don't use SDL event watcher to handle resizes as our redraw is too
	// slow to provide responsive size changes.
	//
	// For handling input, there appears to be marginal to no benefit over using
	// WaitEvents.

	initAudio()

	initGamepad()

	{
		props, err := sdl.CreateProperties()
		if err != nil {
			log.Fatal(err)
		}
		defer props.Destroy()

		props.SetString(sdl.PROP_WINDOW_CREATE_TITLE_STRING, "Wo̅r̅l̅d̅s̅p̅a̅w̅n̅")
		props.SetBoolean(sdl.PROP_WINDOW_CREATE_WAYLAND_SURFACE_ROLE_CUSTOM_BOOLEAN, true)
		props.SetBoolean(sdl.PROP_WINDOW_CREATE_VULKAN_BOOLEAN, true)
		props.SetBoolean(sdl.PROP_WINDOW_CREATE_RESIZABLE_BOOLEAN, true)
		props.SetBoolean(sdl.PROP_WINDOW_CREATE_HIGH_PIXEL_DENSITY_BOOLEAN, true)
		props.SetNumber(sdl.PROP_WINDOW_CREATE_WIDTH_NUMBER, 1280)
		props.SetNumber(sdl.PROP_WINDOW_CREATE_HEIGHT_NUMBER, 800)

		// TODO: make `window` not global, for the sake of tidier code
		window, err = sdl.CreateWindowWithProperties(props)
		if err != nil {
			log.Fatal(err)
		}
	}
	// swapchain = gpu.NewSwapchain(window)

	go redrawLoop()

	if err := window.SetWindowRelativeMouseMode(true); err != nil {
		slog.Warn("failed to enable relative mouse mode on the main window", "err", err)
	}

	slog.Info("gamepads", "gamepads", sdl.GetGamepads())

	// TODO: open all gamepads we have here
	gamepad, _ = sdl.OpenGamepad(sdl.GetGamepads()[0])

	slog.Info("gamepad", "gamepad", gamepad)

	raddr := flag.Arg(0)

	// TODO: should newRemoteSession do the logging instead? Yes.

	session, err := newClient(clientRenderer, raddr)
	if err != nil {
		log.Fatal(err)
	}

	currentSession.Store(session)

	for {
		event, err := sdl.WaitEvent()
		if err != nil {
			log.Fatalln("WaitEvent failed", err)
		}

		handleEvent(event)
	}
}

// TODO: move these things somewhere and/or make them more contextual

var swapchain *gpu.Swapchain
var currentExtent gpu.Int3

func resize(width, height int) {
	if width <= 0 || height <= 0 {
		return // minimized, do nothing
	}

	redrawMu.Lock()
	defer redrawMu.Unlock()

	slog.Info("resize", "width", width, "height", height)

	currentExtent = gpu.Int3{width, height, 1}

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

	// Redraw a single frame, blocking the message loop, so that we
	// don't get another resize without there being something for
	// the user to see.
	//
	// We don't care if this redraw fails. In such case, the redraw
	// loop will be suspended again.
	redrawLocked()

	resizeCond.Broadcast()
}

func redrawLoop() {
	for {
		redraw()
	}
}

func redraw() {
	defer trace.StartRegion(context.Background(), "Redraw").End()

	redrawMu.Lock()
	defer redrawMu.Unlock()

	if !redrawLocked() {
		// Swapchain was out of date, wait for resize.
		resizeCond.Wait()
	}
}

/*
func printOps(ops_ *op.Ops) {
	var r ops.Reader
	r.Reset(&ops_.Internal)
	for {
		encOp, ok := r.Decode()
		if !ok {
			break
		}
		log.Printf("%s %#v", ops.OpType(encOp.Data[0]), encOp)

		switch ops.OpType(encOp.Data[0]) {
		case ops.TypePath:
		}
		// log.Printf("%#v", encOp)
	}
}
*/

// TODO: rename
var myRenderer renderer.Renderer

var swapchainImage *gpu.Image

// TODO: most gui things should be defined by the game code
type pauseMenu struct {
}

// Must be called with redrawMu held.
func redrawLocked() bool {
	defer trace.StartRegion(context.Background(), "Redraw Locked").End()

	if swapchain == nil {
		return false
	}

	var jq gpu.JobQueue

	if swapchainImage == nil {
		swapchainImage = gpu.NewImage(&gpu.ImageConfig{
			Dim:       gpu.ImageDim2D,
			Extent:    currentExtent,
			Layers:    1,
			MipLevels: 1,
			Samples:   1,
			Format:    vk.FORMAT_R8G8B8A8_UNORM,
			Usage:     gpu.ImageUsageLoadStore | gpu.ImageUsageAttachment,
		})
	}

	swapchainImage.EnqueueInit(&jq)
	// defer cq.Garbage(swapchainImage.Destroy)

	// swapchainImage := swapchain.Image(swapchainImageIndex)

	conf := config.Load()

	t := 1.0
	func() {
		clientRenderer.sceneMu.Lock()
		defer clientRenderer.sceneMu.Unlock()

		copy(clientRenderer.privateScene.Parent, clientRenderer.scene.Parent)
		copy(clientRenderer.privateScene.TransformT0, clientRenderer.scene.TransformT0)
		copy(clientRenderer.privateScene.TransformT1, clientRenderer.scene.TransformT1)
		copy(clientRenderer.privateScene.Instance, clientRenderer.scene.Instance)
		copy(clientRenderer.privateScene.MeshInstance, clientRenderer.scene.MeshInstance)
		copy(clientRenderer.privateScene.Pose, clientRenderer.scene.Pose)
		clientRenderer.privateScene.Sky = clientRenderer.scene.Sky
		clientRenderer.privateScene.OurCamera = clientRenderer.scene.OurCamera

		if !conf.DontInterpolate {
			t = min(max(float64(sdl.TicksNS()-clientRenderer.t0sdl)/float64(clientRenderer.t1sdl-clientRenderer.t0sdl), 0), 1)
		}
	}()

	/*
		// TODO: do deformations in their own JobQueues
		var wg gpu.WaitGroup
		wg.Add(len(clientRenderer.privateScene.Pose))
		for i, pose := range clientRenderer.privateScene.Pose {
			if pose == nil {
				continue
			}

			jq := jq.Fork()

			dpose := gpu.MakeSliceUncached[geometry.Mat4x4](len(pose))
			defer jq.Cleanup(func() { gpu.Free(gpu.UnsafePointer(gpu.SliceData(dpose))) })

			dposehost := dpose.Value()
			for i, xform := range pose {
				// TODO: actually let the renderer interpolate
				dposehost[i] = xform
			}

			clientRenderer.privateScene.MeshInstance[i].Mesh.EnqueueDeform(&jq, dpose)

			wg.EnqueueDone(&jq)
		}
		wg.EnqueueWait(&jq)
	*/

	clientRenderer.privateScene.OurCamera.Transform = clientRenderer.privateScene.Transform(1999, float32(t))

	clientRenderer.privateScene2.EnqueueUpdate(&jq, clientRenderer.privateScene, float32(t))

	testTexture := texture("Editor/measure2.ktx2")

	myRenderer.Render(&jq,
		clientRenderer.privateScene2, float32(t),
		&clientRenderer.privateScene.OurCamera,
		swapchainImage, currentExtent,
		testTexture)

	// TODO: it would probably be a good idea to inject overlay rendering into
	// Render so that we can avoid breaking the render pass.

	// Menu

	/*
		var guiOps op.Ops
		func() {
			theme := material.NewTheme()

			gtx := layout.Context{
				Ops: &guiOps,
				Now: time.Now(),
				Metric: unit.Metric{
					PxPerDp: 1,
					PxPerSp: 1,
				},
				Constraints: layout.Exact(image.Pt(currentExtent.X, currentExtent.Y)),
			}

			a := material.H3(theme, "42")

			layout.SE.Layout(gtx, a.Layout)
		}()
	*/

	func() {
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
				RenderArea: vk.Rect2D{Extent: vk.Extent2D{Width: uint32(currentExtent.X), Height: uint32(currentExtent.Y)}},
				LayerCount: 1,
			})
		defer rp.End()

		rp.SetPrimitiveTopology(vk.PRIMITIVE_TOPOLOGY_TRIANGLE_LIST)
		rp.SetPrimitiveRestartEnable(false)

		rp.SetViewports([]vk.Viewport{
			{
				X:        0,
				Y:        0,
				Width:    float32(currentExtent.X),
				Height:   float32(currentExtent.Y),
				MinDepth: 0,
				MaxDepth: 1,
			},
		})
		rp.SetScissors([]vk.Rect2D{
			{Extent: vk.Extent2D{Width: uint32(currentExtent.X), Height: uint32(currentExtent.Y)}},
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

		rp.SetShader(vk.SHADER_STAGE_VERTEX_BIT, testVertMain())
		rp.SetShader(vk.SHADER_STAGE_FRAGMENT_BIT, testFragMain())

		sampler := gpu.NewSampler(&vk.SamplerCreateInfo{
			SType:            vk.STRUCTURE_TYPE_SAMPLER_CREATE_INFO,
			MinFilter:        vk.FILTER_LINEAR,
			MagFilter:        vk.FILTER_LINEAR,
			MipmapMode:       vk.SAMPLER_MIPMAP_MODE_LINEAR,
			AddressModeU:     vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
			AddressModeV:     vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
			AddressModeW:     vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
			MipLodBias:       0.0,
			AnisotropyEnable: vk.FALSE,
			MinLod:           0.0,
			MaxLod:           vk.LOD_CLAMP_NONE,
		})
		defer rp.Cleanup(sampler.Destroy)

		// messagePrinter.Sprintf("New game")
	}()

	var presentationOk bool
	trace.WithRegion(context.Background(), "Presentation", func() {
		presentationOk = swapchain.Present2(&jq, swapchainImage)
	})

	// Only allow one frame-in-flight for now
	//
	// TODO: do multiple frames-in-flight.

	jq.WaitForIdle()

	return presentationOk
}

var clientRenderer = &idk{
	privateScene:  renderer.NewSceneDirty(10000),
	privateScene2: renderer.NewScene(10000),

	scene: renderer.NewSceneDirty(10000),
}

func readSamples(r io.Reader, format int) ([]float32, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	switch format {
	case 1:
		bufSNORM16 := unsafe.Slice((*int16)(unsafe.Pointer(unsafe.SliceData(buf))), len(buf)/2)
		bufFLOAT32 := make([]float32, len(bufSNORM16))
		for i := range bufFLOAT32 {
			bufFLOAT32[i] = max(float32(bufSNORM16[i])/32767.0, -1)
		}
		return bufFLOAT32, nil

	case 2:
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

// TODO: rename to smth like rendererGlue? idk
type idk struct {
	// TODO: remove privateScen and only keep privateScene2. Enqueue updates to
	// privateScene2 directly from scene.

	privateScene  *renderer.SceneDirty // TODO: remove
	privateScene2 *renderer.Scene      // gpu's private copy of the scene; TODO: rename

	sceneMu        sync.Mutex // TODO: rename, embed or leave as is?
	scene          *renderer.SceneDirty
	t0sdl, t1sdl   uint64 // TODO: special type to represent SDL ticks?
	t0game, t1game worldspawn.Time
}

func (sr *idk) Tick(w *worldspawn.World, playerID ecs.ID, t0, t1 worldspawn.Time, frameDuration time.Duration) {
	sr.sceneMu.Lock()
	defer sr.sceneMu.Unlock()

	conf := config.Load()

	{
		sr.t0sdl = sdl.TicksNS()
		sr.t1sdl = sr.t0sdl + uint64(frameDuration) // depends on timescale

		sr.t0game = t0
		sr.t1game = t1

		for i := range sr.scene.Parent {
			sr.scene.Parent[i] = 0
		}

		sr.scene.TransformT0, sr.scene.TransformT1 = sr.scene.TransformT1, sr.scene.TransformT0
		for i := range sr.scene.Instance {
			sr.scene.Instance[i].Transform = 0
		}
		for i := range sr.scene.Pose {
			sr.scene.Pose[i] = nil
		}
		sr.scene.Sky = texture(w.Sky).View

		playerEntity, _ := w.Entity.Load(playerID)

		// TODO: we should just parent stuff to bone
		activeWeapon := playerEntity.(worldspawn.FPSCharacter).ActiveWeapon

		for id, v := range ecs.Join(w.RendererModel, w.TranslationRotation) {
			rendererModel := v.V1
			positionRotation := v.V2
			scale, ok := w.Scale.Load(id)
			if !ok {
				scale = geometry.Vec3Broadcast(1)
			}
			cosmeticOffset, _ := w.CosmeticOffset.Load(id)

			i := id.Index()

			mask := uint8(0b11)

			if id == playerID {
				mask = 0b10
			}

			// We'll probably want extra code for figuring out where to draw/play
			// sound of certain entities instead of having dedicated viewmodel
			// components
			if id == activeWeapon {
				viewmodel, _ := w.Viewmodel.Load(id)
				aim, _ := w.WeaponAim.Load(id)

				// TODO: should we use a pivot point for viewmodel?

				positionRotation, _ = w.TranslationRotation.Load(playerID)
				velocity, _ := w.Velocity.Load(playerID)

				player, _ := w.Entity.Load(playerID)
				fpsCharacter := player.(worldspawn.FPSCharacter)

				up := positionRotation.Rotation.Rotate(geometry.Vec3{0, 0, 1})

				rot := aim.ShootRotation.
					Mul(geometry.Rot3FromPlaneAngle(geometry.Vec3{0, -1, 0}, float32(0)))

				_ = fpsCharacter
				_ = rot

				horizontalVelocity := velocity.Linear.Add(up.Scale(-velocity.Linear.Dot(up)))

				// worldspawn.Time{} here is an ugly hack, stop doing that
				viewmodelSway := geometry.Vec3{0, float32(math.Sin(10*durationSeconds(w.Now.Sub(worldspawn.Time(0)))) * 0.01 * min(float64(horizontalVelocity.Length()), 1)), 0}

				sr.scene.Parent[i] = 1999

				positionRotation.Translation = geometry.DVec3FromVec3(viewmodel.Translation.Add(viewmodelSway))
				positionRotation.Rotation = geometry.Rot3One()

				// I don't like the idea of changing the scale for viewmodels to
				// be honest, so let's just not
				scale = geometry.Vec3Broadcast(1)

				mask = 0b01
			}

			var offset geometry.Vec3
			if !conf.DisableCosmeticOffset {
				offset = cosmeticOffset.Evaluate(w.Now)
			}

			sr.scene.TransformT1[i] = geometry.TRS3{
				Translation: positionRotation.Translation.Add(geometry.DVec3FromVec3(offset)).Vec3(),
				Rotation:    positionRotation.Rotation,
				Scale:       scale,
			}

			sr.scene.MeshInstance[i].Mesh = model(rendererModel.Filename)

			/*
				// This is just horribly broken

				baseMesh := model(rendererModel.Filename)

				// tbf we can always just allocate a unique deforming mesh
				mesh := sr.scene.MeshInstance[i].Mesh

				if sr.scene.MeshInstance[i].BaseMesh != baseMesh {
					mesh = new(renderer.Mesh)
					mesh.InitDeforming(baseMesh)
					sr.scene.MeshInstance[i].BaseMesh = baseMesh
					sr.scene.MeshInstance[i].Mesh = mesh
				}

				if pose, ok := w.Pose.Load(id); ok {
					indirectedPose := make([]geometry.Mat4x4, len(mesh.VertexGroups))
					for i, groupName := range mesh.VertexGroups {
						xform, ok := pose[groupName]
						if !ok {
							xform = geometry.Mat4x4Identity()
						}
						indirectedPose[i] = xform
					}
					sr.scene.Pose[i] = indirectedPose
				}
			*/

			sr.scene.Instance[i].Mask = mask

			sr.scene.Instance[i].Transform = i
		}

		i := 1999
		{
			playerEntity, _ := w.Entity.Load(playerID)
			fpsCharacter := playerEntity.(worldspawn.FPSCharacter)
			positionRotation, _ := w.TranslationRotation.Load(playerID)
			viewPunch, ok := w.ViewPunch.Load(playerID)
			if !ok {
				viewPunch = geometry.Rot3One()
			}

			sr.scene.TransformT1[i] = geometry.TRS3{
				Translation: positionRotation.Translation.Add(geometry.DVec3FromVec3(positionRotation.Rotation.Rotate(geometry.Vec3{0, 0, fpsCharacter.StandingViewHeight}))).Vec3(),
				Rotation: positionRotation.Rotation.
					Mul(geometry.Rot3FromPlaneAngle(geometry.Vec3{0, 0, -1}, 2*math.Pi*fpsCharacter.Look.X)).
					Mul(geometry.Rot3FromPlaneAngle(geometry.Vec3{-1, 0, 0}, 2*math.Pi*fpsCharacter.Look.Y)).
					Mul(viewPunch),
				Scale: geometry.Vec3Broadcast(1),
			}

			// TODO: should we change FieldOfView to FocalLength? It potentially
			// might be more intuitive, such as when adding or multiplying.
			// Alternatively we could just use InverseFieldOfView or some other
			// thing with comparable properties but not ambiguous (FocalLength
			// depends on sensor size which is commonly understood to be 35 mm)
			sr.scene.OurCamera = renderer.Camera{
				FieldOfView:   float32(geometry.Radians(67.5)),
				NearClipPlane: 0.01,
			}
		}
	}

	{
		var scene sfx.Scene
		scene.TransformT0 = []geometry.TRS3{{}} // 0th transform is reserved
		scene.Instance = []sfx.Instance{}

		for id, soundEffect := range w.SoundEffect.All() {
			positionRotation, _ := w.TranslationRotation.Load(id)
			scale, _ := w.Scale.Load(id)

			scene.TransformT0 = append(scene.TransformT0, geometry.TRS3{
				Translation: positionRotation.Translation.Vec3(), // TODO: we should also be applying cosmetic offset like in rendering
				Rotation:    positionRotation.Rotation,
				Scale:       scale,
			})

			effect, ok := sources[soundEffect.Effect]
			if !ok {
				f, err := worldspawn.Data.Open(soundEffect.Effect)
				if err != nil {
					// TODO: should be non-fatal
					panic(fmt.Sprintf("failed to open file %v", soundEffect.Effect))
				}

				switch path.Ext(soundEffect.Effect) {
				case ".wav":
					reader, _ := wav.NewReader(f.(io.ReaderAt))
					// TODO: wav should implement normal io.Reader tbh.
					samples, _ := readSamples(io.NewSectionReader(reader, 0, math.MaxInt64), reader.Format())
					effect = &sfx.Source{
						Samples:    extractChannel(samples, reader.Channels(), 0),
						SampleRate: reader.SampleRate(),
					}

				case ".opus":
					reader, _ := opusfile.NewReader(f)
					samples, _ := readSamples(reader, 2)
					effect = &sfx.Source{
						Samples:    extractChannel(samples, reader.Channels(), 0),
						SampleRate: reader.SampleRate(),
					}

				default:
					panic("unsupported")
				}

				sources[soundEffect.Effect] = effect
			}

			scene.Instance = append(scene.Instance, sfx.Instance{
				Transform: len(scene.TransformT0) - 1,
				Source:    effect,
				PlayTime:  soundEffect.PlayTime.Sub(worldspawn.Time(0)),
			})
		}

		// TODO: let us do multiple audio renders per frame. Should be nice for
		// sessions with long ticks
		renderAudio(scene, t0.Sub(worldspawn.Time(0)), t1.Sub(t0))
	}
}

func (sr *idk) Subtick(w *worldspawn.World, playerID ecs.ID) {
	sr.sceneMu.Lock()
	defer sr.sceneMu.Unlock()

	i := 1999

	playerEntity, _ := w.Entity.Load(playerID)
	fpsCharacter := playerEntity.(worldspawn.FPSCharacter)
	positionRotation, _ := w.TranslationRotation.Load(playerID)
	viewPunch, ok := w.ViewPunch.Load(playerID)
	if !ok {
		viewPunch = geometry.Rot3One()
	}

	rot := positionRotation.Rotation.
		Mul(geometry.Rot3FromPlaneAngle(geometry.Vec3{0, 0, -1}, 2*math.Pi*fpsCharacter.Look.X)).
		Mul(geometry.Rot3FromPlaneAngle(geometry.Vec3{-1, 0, 0}, 2*math.Pi*fpsCharacter.Look.Y)).
		Mul(viewPunch) // doing this here causes judder, we should continue interpolating viewPunch somehow. Perhaps add an extra entity or transform in the renderer's transform hierarchy..?

	sr.scene.TransformT0[i].Rotation = rot
	sr.scene.TransformT1[i].Rotation = rot
}

func step(edge, x float32) float32 {
	if x < edge {
		return 0
	} else {
		return 1
	}
}

// should be moved to worldspawn.*
// TODO: should append commands instead of constructing an entire InputCommands
// object
// TODO: should take the time
func handleInput(cmds *[]worldspawn.InputCmd2, t worldspawn.Time, action int32, value float32) {
	/*
		cmd := worldspawn.InputPacket{
			Time: t,
		}
	*/

	switch action {
	case worldspawn.ActionJump:
		if value != 0 {
			*cmds = append(*cmds, worldspawn.InputCmd2{Time: t, Cmd: worldspawn.ButtonDown(worldspawn.ButtonJump)})
		} else {
			*cmds = append(*cmds, worldspawn.InputCmd2{Time: t, Cmd: worldspawn.ButtonUp(worldspawn.ButtonJump)})
		}

	case worldspawn.ActionAttack:
		if value != 0 {
			*cmds = append(*cmds, worldspawn.InputCmd2{Time: t, Cmd: worldspawn.ButtonDown(worldspawn.ButtonAttack)})
		} else {
			*cmds = append(*cmds, worldspawn.InputCmd2{Time: t, Cmd: worldspawn.ButtonUp(worldspawn.ButtonAttack)})
		}

	case worldspawn.ActionReload:
		if value != 0 {
			*cmds = append(*cmds, worldspawn.InputCmd2{Time: t, Cmd: worldspawn.ButtonDown(worldspawn.ButtonReload)})
		} else {
			*cmds = append(*cmds, worldspawn.InputCmd2{Time: t, Cmd: worldspawn.ButtonUp(worldspawn.ButtonReload)})
		}

	case worldspawn.ActionSlot0:
		*cmds = append(*cmds, worldspawn.InputCmd2{Time: t, Cmd: worldspawn.Slot(0)})

	case worldspawn.ActionSlot1:
		*cmds = append(*cmds, worldspawn.InputCmd2{Time: t, Cmd: worldspawn.Slot(1)})

	case worldspawn.ActionSlot2:
		*cmds = append(*cmds, worldspawn.InputCmd2{Time: t, Cmd: worldspawn.Slot(2)})

	case worldspawn.ActionSlot3:
		*cmds = append(*cmds, worldspawn.InputCmd2{Time: t, Cmd: worldspawn.Slot(3)})

	case worldspawn.ActionMoveX:
		*cmds = append(*cmds, worldspawn.InputCmd2{Time: t, Cmd: worldspawn.MoveX(value)})

	case worldspawn.ActionMoveY:
		*cmds = append(*cmds, worldspawn.InputCmd2{Time: t, Cmd: worldspawn.MoveY(-value)})

	case worldspawn.ActionDLookX:
		*cmds = append(*cmds, worldspawn.InputCmd2{Time: t, Cmd: worldspawn.DLookX(value)})

	case worldspawn.ActionDLookY:
		*cmds = append(*cmds, worldspawn.InputCmd2{Time: t, Cmd: worldspawn.DLookY(value)})
	}
}

// var gamepadCmds []any

type flickStick struct {
	activated      bool
	deflection     geometry.Vec2
	lastDeflection geometry.Vec2
}

var flickStickTest flickStick

func sdlTimeToGameTime(ticks uint64) worldspawn.Time {
	clientRenderer.sceneMu.Lock()
	defer clientRenderer.sceneMu.Unlock()

	// If we have DontInterpolate set, we'll want t = 1
	t := min(max(float64(ticks-clientRenderer.t0sdl)/float64(clientRenderer.t1sdl-clientRenderer.t0sdl), 0), 1)

	return clientRenderer.t0game.Add(time.Duration(float64(clientRenderer.t1game-clientRenderer.t0game) * t))
}

func handleEvent(event any) {
	var cmds []worldspawn.InputCmd2

	switch event := event.(type) {
	case *sdl.QuitEvent:
		doExit()

	case *sdl.WindowPixelSizeChangedEvent:
		resize(int(event.Data1), int(event.Data2))

	case *sdl.KeyDownEvent:
		inputMu.Lock()
		defer inputMu.Unlock()

		// TODO: respect WindowID
		// TODO: respect Which

		actionSet := actionSets["ON_FOOT"]

		actionName := actionSet.Keys[event.Key]

		handleInput(&cmds, sdlTimeToGameTime(event.Timestamp), actionName, 1)

	case *sdl.KeyUpEvent:
		inputMu.Lock()
		defer inputMu.Unlock()

		actionSet := actionSets["ON_FOOT"]

		actionName := actionSet.Keys[event.Key]

		handleInput(&cmds, sdlTimeToGameTime(event.Timestamp), actionName, 0)

	case *sdl.MouseMotionEvent:
		inputMu.Lock()
		defer inputMu.Unlock()

		// TODO: respect WindowID
		// TODO: respect Which

		eventTime := sdlTimeToGameTime(event.Timestamp)

		handleInput(&cmds, eventTime, worldspawn.ActionDLookX, event.XRel*0.0005)
		handleInput(&cmds, eventTime, worldspawn.ActionDLookY, event.YRel*0.0005)

	case *sdl.MouseButtonDownEvent:
		inputMu.Lock()
		defer inputMu.Unlock()

	case *sdl.MouseButtonUpEvent:
		inputMu.Lock()
		defer inputMu.Unlock()

	case *sdl.GamepadAxisMotionEvent:
		inputMu.Lock()
		defer inputMu.Unlock()

		value := max(float32(event.Value)/32767, -1)

		actionSet := actionSets["ON_FOOT"]

		eventTime := sdlTimeToGameTime(event.Timestamp)

		switch event.Axis {
		case sdl.GAMEPAD_AXIS_LEFTX:
			if math.Abs(float64(value)) < 0.2 {
				value = 0
			}
			handleInput(&cmds, eventTime, worldspawn.ActionMoveX, value)
		case sdl.GAMEPAD_AXIS_LEFTY:
			if math.Abs(float64(value)) < 0.2 {
				value = 0
			}
			handleInput(&cmds, eventTime, worldspawn.ActionMoveY, value)
		case sdl.GAMEPAD_AXIS_RIGHTX:
			flickStickTest.deflection.X = value
		case sdl.GAMEPAD_AXIS_RIGHTY:
			flickStickTest.deflection.Y = value
		case sdl.GAMEPAD_AXIS_RIGHT_TRIGGER:
			handleInput(&cmds, eventTime, actionSet.RightTrigger, value)
			handleInput(&cmds, eventTime, actionSet.RightTriggerFullPull, step(0.9, value))
		}

	case *sdl.GamepadButtonDownEvent:
		inputMu.Lock()
		defer inputMu.Unlock()

		actionSet := actionSets["ON_FOOT"]

		actionName := actionSet.Buttons[sdl.GamepadButton(event.Button)]

		handleInput(&cmds, sdlTimeToGameTime(event.Timestamp), actionName, 1)

	case *sdl.GamepadButtonUpEvent:
		inputMu.Lock()
		defer inputMu.Unlock()

		actionSet := actionSets["ON_FOOT"]

		actionName := actionSet.Buttons[sdl.GamepadButton(event.Button)]

		handleInput(&cmds, sdlTimeToGameTime(event.Timestamp), actionName, 0)

	case *sdl.GamepadUpdateCompleteEvent:
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

			handleInput(&cmds, sdlTimeToGameTime(event.Timestamp), worldspawn.ActionDLookX, dlookx)

			flickStickTest.lastDeflection = flickStickTest.deflection
		}
	}

	if len(cmds) > 0 {
		session := currentSession.Load()
		session.HandleInput(cmds)
	}
}

// TODO: rename
func doExit() {
	// TODO: we should shut down audio properly to avoid audio artifacts on
	// pipewire

	os.Exit(0)
}

func mustReadFile(filename string) []byte {
	data, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return data
}
