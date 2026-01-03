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
	tm timeMapping

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
	for ; i != -1; i = s.Parent[i] {
		A := s.TransformT0[i].NLerp(s.TransformT1[i], t).Mat4x4()
		B = A.Mul4x4(B)
	}
	return B
}

type timeMapping struct {
	t0sdl, t1sdl   uint64 // TODO: special type to represent SDL ticks?
	t0game, t1game game.Time
}

type renderer struct {
	lastGen       []uint32
	lastTransform []geometry.TRS3

	// The update that didn't fit into the queue
	stagingUpdate *sceneUpdate
	// Queue of updates, consumed by Render
	updates chan *sceneUpdate

	stuffMu sync.Mutex
	tm      timeMapping
	// TODO: what if we want to pass multiple cameras to the composition
	// pipeline?
	// TODO: camera states need to be t0 and t1 too
	ourCamera          pathtracer.Camera
	ourCameraTransform int
	fn                 uint32
	scene2             *sceneUpdate
	scene              *pathtracer.Scene

	// Uhh
	sfxScene *sfx.Scene
}

// TODO: remove this in favor of merging updates at commitUpdate time. I.e.
// we'll start off with a clean update every time.
func (re *renderer) beginUpdate() *sceneUpdate {
	if re.stagingUpdate == nil {
		// TODO: pool this stuff
		return newSceneDirty(10000)
	}
	tmp := re.stagingUpdate
	re.stagingUpdate = nil
	return tmp
}

// TODO: rename to enqueueUpdate?
func (re *renderer) commitUpdate(update *sceneUpdate) {
	select {
	case re.updates <- update:
	default:
	}
}

func (re *renderer) Tick(w *game.Scene, playerID ecs.ID, t0, t1 game.Time, frameDuration time.Duration) {
	conf := config.Load()

	player, _ := w.Entity.Get(playerID)
	fpsCharacter, _ := player.(game.Player)

	update := re.beginUpdate()
	defer re.commitUpdate(update)

	{
		for i := range update.Parent {
			update.Parent[i] = -1
		}

		update.Sky = texture(w.Globals().Sky).Image

		for id := range ecs.All(&w.LocalTranslationRotation) {
			cosmeticOffset, _ := w.CosmeticOffset.Get(id)

			i := id.Index()

			if parent := w.GetParent(id); parent != 0 {
				update.Parent[i] = parent.Index()
			}

			var offset geometry.Vec3
			if !conf.Developer.DisableCosmeticOffset {
				offset = cosmeticOffset.Eval(w.Now)
			}

			tmp, _ := w.GetLocalTRS(id)
			// TODO: we should not record cosmetic offset into renderer.transformT0
			transformT1 := geometry.TRS3{geometry.ConvertVec3[float32](tmp.T).Add(offset), tmp.R, tmp.S}

			transformT0 := re.lastTransform[i]
			if re.lastGen[i] != id.Generation() {
				transformT0 = transformT1
			}

			update.TransformT0[i] = transformT0
			update.TransformT1[i] = transformT1

			re.lastGen[i] = id.Generation()
			re.lastTransform[i] = transformT1
		}

		// TODO: we need to split operations on caches into probe and fetch, so
		// that when a probe fails, we spawn a new goroutine with fetch and
		// everything that follows.

		for id, v := range ecs.Join(&w.RenderingGeometry, &w.LocalTranslationRotation) {
			renderingGeometry := v.V1

			i := id.Index()

			mask := uint8(0b11)
			if visibility, ok := w.Visibility.Get(id); ok {
				switch visibility.Mode {
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
			update.Materials[i] = make([]*pathtracer.InterpretedMaterial, len(mesh.materials))
			update.MaterialArgs[i] = make([][256]byte, len(mesh.materials))

			for j := range update.Materials[i] {
				m2 := getmaterial(mesh.materials[j])
				update.Materials[i][j] = m2.material
				m2.preamble(update.MaterialArgs[i][j][:], &matPropReader{w, id})
			}
		}

		t0sdl := sdl.TicksNS()
		update.tm = timeMapping{
			t0sdl:  t0sdl,
			t1sdl:  t0sdl + uint64(frameDuration),
			t0game: t0,
			t1game: t1,
		}
		update.camera = pathtracer.Camera{
			FieldOfView:   float32(geometry.Radians(67.5)),
			NearClipPlane: 0.01,
		}
		update.cameraTransform = fpsCharacter.Camera.Index()
	}

	// Ughhhhhhh
	{
		camera := update.Transform(fpsCharacter.Camera.Index(), 0)
		cameraPos := geometry.Vec3{camera[0][3], camera[1][3], camera[2][3]}

		scene := re.sfxScene

		a := int64(t1.Sub(t0) * 48000 / 1e9)

		clear(scene.Instance)

		for id, soundEffect := range ecs.All(&w.SoundEffect) {
			trs, _ := w.GetGlobalTRS(id)

			xform := geometry.TRS3{
				T: geometry.ConvertVec3[float32](trs.T), // TODO: we should also be applying cosmetic offset like in video
				R: trs.R,
				S: trs.S,
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
				Transform:   xform,
				Samples:     effect.Samples,
				Attenuation: soundEffect.Attenuation,
				PlayTime:    int64(soundEffect.PlayTime.Sub(game.Time(0)) * 48000 / 1e9),
			}
		}

		// TODO: let us do multiple audio renders per frame. Should be nice for
		// sessions with long ticks
		renderAudio(re.sfxScene, cameraPos, int64(t0.Sub(game.Time(0))*48000/1e9), a)
	}
}

func (re *renderer) Subtick(w *game.Scene, playerID ecs.ID) {
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

func (re *renderer) Render(jq *gpu.JobQueue, sdlNow uint64, dst *gpu.Image) {
	conf := config.Load()

	select {
	case update := <-re.updates:
		re.stuffMu.Lock()
		re.tm = update.tm
		re.stuffMu.Unlock()
		re.ourCamera = update.camera
		re.ourCameraTransform = update.cameraTransform
		re.scene2 = update
		re.scene.SetSky(update.Sky)
		for i := range update.Mask {
			re.scene.SetInstanceGeometry(i, update.Mask[i], update.Mesh[i], update.Materials[i], update.MaterialArgs[i])
		}
	default:
	}

	t := 1.0
	if !conf.Developer.DontInterpolate {
		// TODO: we need to be able to lock sceneMu for this but we can't.
		// We should make our own scene type with the timestamps and stuff.
		t = min(max(float64(sdlNow-re.tm.t0sdl)/float64(re.tm.t1sdl-re.tm.t0sdl), 0), 1)
	}

	camera := re.ourCamera
	if re.scene2 != nil {
		camera.Transform = re.scene2.Transform(re.ourCameraTransform, float32(t))
		for i := range re.scene2.Mask {
			tmp := re.scene2.Transform(i, float32(t))
			// TODO: outline this
			var tmp2 [3][4]float32
			for i := range tmp2 {
				for j := range tmp2[i] {
					tmp2[i][j] = tmp[i][j]
				}
			}
			re.scene.SetInstanceTransform(i, tmp2)
		}
	}

	re.scene.EnqueueBuildAccel(jq)

	re.scene.Render(
		jq,
		pathtracer.Film{
			Extent: dst.Extent(),
			Color:  dst,
		},
		re.fn,
		&camera,
		&pathtracer.Quality{
			MaxBounces:               conf.Quality.MaxBounces,
			RussianRouletteThreshold: conf.Quality.RussianRouletteThreshold,
		})
	re.fn++
}

type matPropReader struct {
	scene  *game.Scene
	object ecs.ID
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
