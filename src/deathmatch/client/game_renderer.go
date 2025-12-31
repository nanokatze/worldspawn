package main

import (
	"fmt"
	"io"
	"path"
	"reflect"
	"sync"
	"time"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/geometry-go"
	"worldspawn/gpu"
	"worldspawn/internal/ecs"
	sfx "worldspawn/internal/fuckwwise"
	"worldspawn/internal/fuckwwise/opusfile"
	"worldspawn/internal/fuckwwise/wav"
	"worldspawn/internal/pathtracer"
	"worldspawn/sdl"
)

type sceneUpdate struct {
	t0sdl, t1sdl   uint64 // TODO: special type to represent SDL ticks?
	t0game, t1game game.Time

	camera          pathtracer.Camera
	cameraTransform int

	Sky *gpu.Image

	Parent      []int
	TransformT0 []geometry.TRS3
	TransformT1 []geometry.TRS3
	// TODO: we also need to carry velocity here for motion blur, or at least
	// some extra info to disambiguate fast temporally-aliased motions.

	Mask         []uint8
	Mesh         []*pathtracer.Mesh
	Materials    [][]*pathtracer.InterpretedMaterial
	MaterialArgs [][][256]byte
}

func newSceneDirty(n int) *sceneUpdate {
	return &sceneUpdate{
		Parent:      make([]int, n),
		TransformT0: make([]geometry.TRS3, n),
		TransformT1: make([]geometry.TRS3, n),

		Mask:         make([]uint8, n),
		Mesh:         make([]*pathtracer.Mesh, n),
		Materials:    make([][]*pathtracer.InterpretedMaterial, n),
		MaterialArgs: make([][][256]byte, n),
	}
}

// TODO: rename to something like GlobalTransform?
func (s *sceneUpdate) Transform(i int, t float32) geometry.Mat4x4 {
	B := geometry.Mat4x4Identity()
	for ; i != 0; i = s.Parent[i] {
		A := s.TransformT0[i].NLerp(s.TransformT1[i], t).Mat4x4()
		B = A.Mul4x4(B)
	}
	return B
}

type gameRendererImpl struct {
	transformT0 []geometry.TRS3

	// The update that didn't fit into the queue
	stagingUpdate *sceneUpdate
	// Queue of updates, consumed by Render
	updates chan *sceneUpdate

	stuffMu        sync.Mutex
	t0sdl, t1sdl   uint64 // TODO: special type to represent SDL ticks?
	t0game, t1game game.Time
	// TODO: what if we want to pass multiple cameras to the composition
	// pipeline?
	ourCamera          pathtracer.Camera
	ourCameraTransform int
	fn                 uint32
	scene2             *sceneUpdate
	scene              *pathtracer.Scene

	// Uhh
	sfxScene *sfx.Scene
}

// TODO: should be created at runtime
var gameRenderer = &gameRendererImpl{
	transformT0: make([]geometry.TRS3, 10000),

	updates: make(chan *sceneUpdate, 1),

	scene: pathtracer.NewScene(10000, 5),

	sfxScene: &sfx.Scene{
		Instance: make([]sfx.Instance, 10000),
	},
}

// TODO: remove this in favor of merging updates at commitUpdate time. I.e.
// we'll start off with a clean update every time.
func (renderer *gameRendererImpl) beginUpdate() *sceneUpdate {
	if renderer.stagingUpdate == nil {
		// TODO: pool this stuff
		return newSceneDirty(10000)
	}
	tmp := renderer.stagingUpdate
	renderer.stagingUpdate = nil
	return tmp
}

// TODO: rename to enqueueUpdate?
func (renderer *gameRendererImpl) commitUpdate(update *sceneUpdate) {
	select {
	case renderer.updates <- update:
	default:
	}
}

func (renderer *gameRendererImpl) Tick(w *game.Scene, playerID ecs.Entity, t0, t1 game.Time, frameDuration time.Duration) {
	conf := config.Load()

	update := renderer.beginUpdate()
	defer renderer.commitUpdate(update)

	{
		for i := range update.Parent {
			update.Parent[i] = 0
		}

		update.Sky = texture(w.Sky()).Image

		playerEntity, _ := w.Entity.Get(playerID)
		fpsCharacter := playerEntity.(game.FPSCharacter)

		for id, tr := range w.TranslationRotation.All() {
			cosmeticOffset, _ := w.CosmeticOffset.Get(id)

			i := id.Index()

			parent, hasParent := w.Parent.Get(id)
			if hasParent {
				update.Parent[i] = parent.Index()
			}

			var offset geometry.Vec3
			if !conf.Developer.DisableCosmeticOffset {
				offset = cosmeticOffset.Eval(w.Now)
			}

			transformT0 := renderer.transformT0[i]
			transformT1 := geometry.TRS3{
				Translation: tr.Translation.Add(geometry.DVec3FromVec3(offset)).Vec3(),
				Rotation:    tr.Rotation,
				Scale:       w.GetScale(id),
			}

			update.TransformT0[i] = transformT0
			update.TransformT1[i] = transformT1
			renderer.transformT0[i] = transformT1
		}

		// TODO: we need to split operations on caches into probe and fetch, so
		// that when a probe fails, we spawn a new goroutine with fetch and
		// everything that follows.

		for id, v := range ecs.Join(w.RenderingGeometry, w.TranslationRotation) {
			renderingGeometry := v.V1

			i := id.Index()

			mask := uint8(0b11)
			if viewmodel, hasViewmodel := w.Viewmodel2.Get(id); hasViewmodel {
				switch viewmodel.Mode {
				case 1:
					mask = 0b01
				case 2:
					mask = 0b10
				}
			}
			update.Mask[i] = mask

			mesh := getmesh(renderingGeometry)

			update.Mesh[i] = mesh.re

			// TODO: stop allocating a new slice every time
			update.Materials[i] = make([]*pathtracer.InterpretedMaterial, len(mesh.defaultMaterials))
			update.MaterialArgs[i] = make([][256]byte, len(mesh.defaultMaterials))

			for j := range update.Materials[i] {
				m2 := getmaterial(mesh.defaultMaterials[j])
				update.Materials[i][j] = m2.material
				m2.preamble(update.MaterialArgs[i][j][:], &matPropReader{w, id})
			}
		}

		update.t0sdl = sdl.TicksNS()
		update.t1sdl = update.t0sdl + uint64(frameDuration)
		update.t0game = t0
		update.t1game = t1
		update.camera = pathtracer.Camera{
			FieldOfView:   float32(geometry.Radians(67.5)),
			NearClipPlane: 0.01,
		}
		update.cameraTransform = fpsCharacter.Camera.Index()
	}

	// Ughhhhhhh
	{
		playerEntity, _ := w.Entity.Get(playerID)
		fpsCharacter := playerEntity.(game.FPSCharacter)
		camera := update.Transform(fpsCharacter.Camera.Index(), 0)
		cameraPos := geometry.Vec3{camera[0][3], camera[1][3], camera[2][3]}

		scene := renderer.sfxScene

		a := int64(t1.Sub(t0) * 48000 / 1e9)

		clear(scene.Instance)

		for id, soundEffect := range w.SoundEffect.All() {
			positionRotation, _ := w.TranslationRotation.Get(id)
			scale := w.GetScale(id)

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
		renderAudio(renderer.sfxScene, cameraPos, int64(t0.Sub(game.Time(0))*48000/1e9), a)
	}
}

func (renderer *gameRendererImpl) Subtick(w *game.Scene, playerID ecs.Entity) {
	// TODO: this will need to enqueue an update and not modify any fields directly!

	// re.stuffMu.Lock()
	// defer re.stuffMu.Unlock()

	// TODO: we'll need to fix camera shenanigans first

	// playerEntity, _ := w.Entity.Get(playerID)
	// fpsCharacter := playerEntity.(game.FPSCharacter)
	// cameraID := fpsCharacter.Camera
	// tr, _ := w.TranslationRotation.Get(cameraID)

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

func (renderer *gameRendererImpl) Render(jq *gpu.JobQueue, sdlNow uint64, dst *gpu.Image) {
	conf := config.Load()

	select {
	case update := <-renderer.updates:
		renderer.stuffMu.Lock()
		renderer.t0sdl = update.t0sdl
		renderer.t1sdl = update.t1sdl
		renderer.t0game = update.t0game
		renderer.t1game = update.t1game
		renderer.stuffMu.Unlock()
		renderer.ourCamera = update.camera
		renderer.ourCameraTransform = update.cameraTransform
		renderer.scene2 = update
		renderer.scene.SetSky(update.Sky)
		for i := range update.Mask {
			renderer.scene.SetInstanceGeometry(i, update.Mask[i], update.Mesh[i], update.Materials[i], update.MaterialArgs[i])
		}
	default:
	}

	t := 1.0
	if !conf.Developer.DontInterpolate {
		// TODO: we need to be able to lock sceneMu for this but we can't.
		// We should make our own scene type with the timestamps and stuff.
		t = min(max(float64(sdlNow-renderer.t0sdl)/float64(renderer.t1sdl-renderer.t0sdl), 0), 1)
	}

	if renderer.scene2 != nil {
		renderer.ourCamera.Transform = renderer.scene2.Transform(renderer.ourCameraTransform, float32(t))
		for i := range renderer.scene2.Mask {
			tmp := renderer.scene2.Transform(i, float32(t))
			// TODO: outline this
			var tmp2 [3][4]float32
			for i := range tmp2 {
				for j := range tmp2[i] {
					tmp2[i][j] = tmp[i][j]
				}
			}
			renderer.scene.SetInstanceTransform(i, tmp2)
		}
	}

	renderer.scene.EnqueueBuildAccel(jq)

	renderer.scene.Render(
		jq,
		pathtracer.Film{
			Extent: dst.Extent(),
			Color:  dst,
		},
		renderer.fn,
		&renderer.ourCamera,
		&pathtracer.Quality{
			MaxBounces:               conf.Quality.MaxBounces,
			RussianRouletteThreshold: conf.Quality.RussianRouletteThreshold,
		})
	renderer.fn++
}

type matPropReader struct {
	scene  *game.Scene
	object ecs.Entity
}

func (sr *matPropReader) UniformAttribute(name string, out *[4]float32) bool {
	objectData, ok := sr.scene.Entity.Get(sr.object)
	if !ok {
		return false
	}
	*out = getv(reflect.ValueOf(objectData).FieldByName(name))
	return true
}

func getv(v reflect.Value) [4]float32 {
	// TODO: allow types to implement a thing to get the value with
	if v.Type() == reflect.TypeFor[[4]float32]() {
		return v.Interface().([4]float32)
	}
	return [4]float32{}
}
