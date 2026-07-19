package main

import (
	_ "embed"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"structs"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"worldspawn/gpu"
	"worldspawn/gpu/image/draw"
	"worldspawn/gpu/vk"
	"worldspawn/gpu/wsi"
	"worldspawn/internal/gmath"
	"worldspawn/internal/loaders/audio/opusfile"
	"worldspawn/internal/renderer"
	"worldspawn/internal/sdl"
)

//go:embed room.bin
var room []byte

func byteslice[T any](s []T) []byte {
	sizeofT := int(unsafe.Sizeof(*new(T)))
	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(s))), len(s)*sizeofT)
}

type mymesh struct {
	gmesh  *renderer.Geometry
	gaccel gpu.BLAS
}

var getmesh = sync.OnceValue(func() *mymesh {
	verts := gpu.MakeSliceUncached[[3]float32](len(room) / (4 * 3))
	copy(byteslice(verts.Value()), room)

	mesh := new(renderer.Geometry)
	mesh.AttributeBuffers = []any{
		renderer.AttributePosition: verts,
	}
	mesh.Parts = []renderer.GeometryPart{
		{
			IndexBuffer: renderer.IndexBufferIdentity(3 * gpu.SliceLen(verts)),
		},
	}

	accelConfig := mesh.AccelConfig()
	accel := gpu.NewBLAS(accelConfig.CalcSizes().Accel)

	jq := new(gpu.JobQueue)
	accel.EnqueueBuild(jq, accelConfig)
	gpu.WaitForIdle(jq)

	return &mymesh{
		gmesh:  mesh,
		gaccel: accel,
	}
})

var blenderToPathTracer = gmath.Mat4x4f32{
	1, 0, 0, 0,
	0, 0, -1, 0,
	0, -1, 0, 0,
	0, 0, 0, 1,
}

var gfxView = sync.OnceValues(func() (*gpu.RayTracingPipeline, gpu.ShaderBindingTable) {
	raygen := gpu.NewGeneralRayTracingShaderGroup(gpu.NewRayTracingFunc(mustReadFile("shaders/experiments_sfx3d_main.spv"), vk.SHADER_STAGE_RAYGEN_BIT_KHR, "gfxView"))

	raygenRecord := gpu.NewUncached[gpu.RayTracingShaderGroupHandle]()
	*raygenRecord.Value() = raygen.Handle()

	pipe := gpu.LinkRayTracingShaderGroups(raygen)

	sbt := gpu.MakeShaderBindingTable(raygenRecord, gpu.Slice[struct{}]{}, gpu.Slice[struct{}]{}, gpu.Slice[struct{}]{})

	return pipe, sbt
})

var tracePipeline = sync.OnceValues(func() (*gpu.RayTracingPipeline, gpu.ShaderBindingTable) {
	raygen := gpu.NewGeneralRayTracingShaderGroup(gpu.NewRayTracingFunc(mustReadFile("shaders/experiments_sfx3d_main.spv"), vk.SHADER_STAGE_RAYGEN_BIT_KHR, "trace"))

	raygenRecord := gpu.NewUncached[gpu.RayTracingShaderGroupHandle]()
	*raygenRecord.Value() = raygen.Handle()

	pipe := gpu.LinkRayTracingShaderGroups(raygen)

	sbt := gpu.MakeShaderBindingTable(raygenRecord, gpu.Slice[struct{}]{}, gpu.Slice[struct{}]{}, gpu.Slice[struct{}]{})

	return pipe, sbt
})

type Vertex struct {
	_           structs.HostLayout
	Emitter     uint32
	T           float32
	Attenuation float32
}

type Path struct {
	_                   structs.HostLayout
	T                   float32
	Throughput          float32
	PropagationVelocity float32
	RayOrigin           gmath.Vec3f32
	RayDir              gmath.Vec3f32
	Vertices            gpu.Slice[Vertex]
}

var geodesic = []gmath.Vec3f32{
	{0, 1, 0},
	{0, -1, 0},
	{1, 0, 0},
	{-1, 0, 0},
	{0, 0, 1},
	{0, 0, -1},
}

func main() {
	if err := sdl.InitSubSystem(sdl.INIT_VIDEO | sdl.INIT_AUDIO); err != nil {
		panic(fmt.Sprintf("failed to initialize SDL video subsystem: %v", err))
	}

	window, err := sdl.CreateWindow(
		sdl.WithBooleanProperty(sdl.PROP_WINDOW_CREATE_VULKAN_BOOLEAN, true),
		sdl.WithBooleanProperty(sdl.PROP_WINDOW_CREATE_HIGH_PIXEL_DENSITY_BOOLEAN, true))
	if err != nil {
		log.Fatal(err)
	}

	window.SetTitle("sfx3d")
	window.SetResizable(true)
	window.SetSize(1600, 900)
	window.SetRelativeMouseMode(true)

	au, err := sdl.OpenAudioDeviceStream(sdl.AUDIO_DEVICE_DEFAULT_PLAYBACK, &sdl.AudioSpec{
		Format:     sdl.AUDIO_F32,
		Channels:   2,
		SampleRate: 48000,
	})
	if err != nil {
		log.Fatal(err)
	}
	au.Device().Resume()

	resized := make(chan struct{}, 1)
	var redrawMu sync.Mutex

	var swapchain *wsi.Swapchain

	mesh := getmesh()

	tlasInstances := gpu.MakeSliceUncached[gpu.BLASInstance](1)
	var tmp gpu.BLASInstance
	tmp.InstanceIDAndMask = 0xff << 24
	tmp.Transform = [3][4]float32{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 1, 0},
	}
	tmp.SetAccel(mesh.gaccel)
	tlasInstances.Value()[0] = tmp

	tlasConfig := &gpu.AccelBuildConfig{
		Type: vk.ACCELERATION_STRUCTURE_TYPE_TOP_LEVEL_KHR,
		Inputs: []gpu.AccelBuildInput{
			&gpu.TLASBuildInputInstances{
				Instances:     gpu.SliceData(tlasInstances),
				InstanceCount: uint32(gpu.SliceLen(tlasInstances)),
			},
		},
	}
	tlas := gpu.NewTLAS(tlasConfig.CalcSizes().Accel)

	jq := new(gpu.JobQueue)
	tlas.EnqueueBuild(jq, tlasConfig)
	gpu.WaitForIdle(jq)

	var inputMu sync.Mutex
	heldKeys := make(map[sdl.Keycode]struct{})
	var move = gmath.Vec3f32{0, 0, 2}
	var look [2]float64
	var useIdentityKernel atomic.Bool

	pathStates := gpu.MakeSliceUncached[Path](len(geodesic))
	pathStatesHost := pathStates.Value()

	maxVertsPerPath := 50

	pathVerts := gpu.MakeSliceUncached[Vertex](maxVertsPerPath * len(geodesic))

	doTrace := func() []float32 {
		inputMu.Lock()
		for i := range pathStatesHost {
			pathStatesHost[i] = Path{
				PropagationVelocity: 345,
				Throughput:          1,
				RayOrigin:           move,
				RayDir:              geodesic[i],
				Vertices:            pathVerts.Slice(maxVertsPerPath*i, maxVertsPerPath*i),
			}
		}
		inputMu.Unlock()

		jq := new(gpu.JobQueue)

		pipe, sbt := tracePipeline()

		push := struct {
			TLAS       gpu.TLAS
			PathStates gpu.Pointer[Path]
		}{
			TLAS:       tlas,
			PathStates: gpu.SliceData(pathStates),
		}
		gpu.EnqueueTraceRays(jq, []int{gpu.SliceLen(pathStates)}, pipe, sbt, &push)

		// TODO: given that our sources are pointwise we also probably want to
		// trace a bunch of rays directly to the emitters. This would still
		// probably be a good idea even with non-pointwise sources.

		gpu.WaitForIdle(jq)

		kernel := make([]float32, 3*4800)

		for i := range pathStatesHost {
			// pathDir := geodesic[i]

			directionalAttenuation := float32(1)

			pathVertices := pathStatesHost[i].Vertices.Value()

			// log.Println(i, pathVertices)

			for _, vert := range pathVertices {
				i := int(48000 * vert.T)
				if i >= len(kernel) {
					break
				}
				kernel[i] = directionalAttenuation * vert.Attenuation
			}
		}

		return kernel
	}

	debussy := func() []float32 {
		f, err := os.Open("debussy.opus")
		if err != nil {
			panic(err)
		}

		r, err := opusfile.NewReader(f)
		if err != nil {
			panic(err)
		}

		log.Println(r.Config().Channels, "channels")

		bytes, err := io.ReadAll(r)
		if err != nil {
			panic(err)
		}

		return unsafe.Slice((*float32)(unsafe.Pointer(unsafe.SliceData(bytes))), len(bytes)/4)
	}()

	startplaying := make(chan struct{})

	go func() {
		<-startplaying

		ticker := time.NewTicker(time.Second / 32)
		for {
			<-ticker.C

			kernel := []float32{1}
			if !useIdentityKernel.Load() {
				kernel = doTrace()
			}

			buffer := make([]float32, 2*48000/32)

			for channel := range 2 {
				for i := range 48000 / 32 {
					var sum float32
					for j := range kernel {
						sum += kernel[j] * debussy[2*i+2*j+channel]
					}
					buffer[2*i+channel] = sum / float32(len(geodesic))
				}
			}

			debussy = debussy[2*48000/32:]

			au.Write(byteslice(buffer))
		}
	}()

	lastframe := time.Now()
	redrawLocked := func() bool {
		Δt := float32(float64(time.Since(lastframe)) / 1e9)
		lastframe = time.Now()

		jq := new(gpu.JobQueue)

		swapchainImageIndex := swapchain.Acquire()
		if swapchainImageIndex == -1 {
			return false
		}

		swapchainImage := swapchain.Image(swapchainImageIndex)
		swapchainImage.EnqueueInit(jq)

		inputMu.Lock()

		movespeed := float32(10)
		if _, ok := heldKeys[sdl.K_LSHIFT]; ok {
			movespeed *= 5
		}

		camera := gmath.TRS3f32{
			T: move,
			R: gmath.Rot3AToB(gmath.Vec3f32{0, 1, 0}, gmath.Vec3f32{1, 0, 0}).Pow(float32(look[0])).
				Mul(gmath.Rot3AToB(gmath.Vec3f32{0, 0, 1}, gmath.Vec3f32{0, 1, 0}).Pow(float32(look[1]))),
			S: gmath.Mat3x3UOne[float32](),
		}
		if _, ok := heldKeys[sdl.K_W]; ok {
			move = move.Add(camera.R.Rotate(gmath.Vec3f32{0, 1, 0}.Scale(movespeed * Δt)))
		}
		if _, ok := heldKeys[sdl.K_A]; ok {
			move = move.Add(camera.R.Rotate(gmath.Vec3f32{-1, 0, 0}.Scale(movespeed * Δt)))
		}
		if _, ok := heldKeys[sdl.K_S]; ok {
			move = move.Add(camera.R.Rotate(gmath.Vec3f32{0, -1, 0}.Scale(movespeed * Δt)))
		}
		if _, ok := heldKeys[sdl.K_D]; ok {
			move = move.Add(camera.R.Rotate(gmath.Vec3f32{1, 0, 0}.Scale(movespeed * Δt)))
		}
		if _, ok := heldKeys[sdl.K_SPACE]; ok {
			move = move.Add(gmath.Vec3f32{0, 0, 1}.Scale(movespeed * Δt))
		}

		proj := gmath.Mat4x4InfinitePerspective(
			float32(gmath.Radians(90)),
			float32(swapchainImage.Extent()[0])/float32(swapchainImage.Extent()[1]),
			0.01)

		viewInverse := camera.Compose().ToMat().Mul(blenderToPathTracer.Inverse())

		inputMu.Unlock()

		pipe, sbt := gfxView()

		push := struct {
			ProjInv gmath.Mat4x4f32
			ViewInv gmath.Mat4x4f32
			TLAS    gpu.TLAS
			Output  gpu.ImageDescriptor
		}{
			ProjInv: proj.Inverse(),
			ViewInv: viewInverse,
			TLAS:    tlas,
			Output:  swapchainImage.Descriptor(),
		}
		gpu.EnqueueTraceRays(jq, swapchainImage.Extent(), pipe, sbt, &push)

		hehe := [4]float32{1, 0, 0, 1}
		if !useIdentityKernel.Load() {
			hehe = [4]float32{0, 1, 0, 1}
		}

		draw.Begin(jq,
			&draw.Config{
				ColorAttachments: []draw.Attachment{
					{
						Image: swapchainImage,

						LoadOp: vk.ATTACHMENT_LOAD_OP_CLEAR,
						ClearValue: [4]uint32{
							math.Float32bits(hehe[0]),
							math.Float32bits(hehe[1]),
							0,
							0x3f800000,
						},
					},
				},
				RenderArea: vk.Rect2D{
					Extent: vk.Extent2D{Width: 128, Height: 128},
				},
				LayerCount: 1,
			}).
			End()

		gpu.WaitForIdle(jq)

		swapchain.Present(jq, swapchainImageIndex)

		return true
	}

	redraw := func() bool {
		redrawMu.Lock()
		defer redrawMu.Unlock()

		return redrawLocked()
	}

	go func() {
		for {
			<-resized
			for redraw() {
			}
		}
	}()

eventLoop:
	for {
		e, err := sdl.WaitEvent()
		if err != nil {
			log.Fatalln("WaitEvent failed", err)
		}

		switch e := e.(type) {
		case *sdl.QuitEvent:
			break eventLoop

		case *sdl.WindowPixelSizeChangedEvent:
			redrawMu.Lock()

			currentExtent := [2]int{int(e.Data1), int(e.Data2)}

			swapchain = wsi.NewSwapchain(&wsi.SwapchainConfig{
				Window:     window,
				ColorSpace: vk.COLOR_SPACE_SRGB_NONLINEAR_KHR,
				ImageConfig: gpu.MakeImageConfig(vk.FORMAT_R8G8B8A8_UNORM, currentExtent[:]).
					WithUsage(vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT).
					WithUsage(vk.IMAGE_USAGE_STORAGE_BIT),
				OldSwapchain: swapchain,
			})

			select {
			case resized <- struct{}{}:
			default:
			}

			redrawMu.Unlock()

		case *sdl.MouseMotionEvent:
			inputMu.Lock()
			look[0] += float64(e.XRel) * 0.002
			look[1] += float64(e.YRel) * 0.002
			inputMu.Unlock()

		case *sdl.KeyDownEvent:
			inputMu.Lock()
			heldKeys[e.Key] = struct{}{}
			inputMu.Unlock()

			if e.Key == sdl.K_X {
				select {
				case startplaying <- struct{}{}:

				default:
					v := !useIdentityKernel.Load()
					useIdentityKernel.Store(v)
				}
			}

		case *sdl.KeyUpEvent:
			inputMu.Lock()
			delete(heldKeys, e.Key)
			inputMu.Unlock()

		default:
			_ = e
		}
	}
}

func mustReadFile(filename string) []byte {
	data, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return data
}
