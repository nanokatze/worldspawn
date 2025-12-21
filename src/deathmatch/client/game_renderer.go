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

	// TODO: remove renderer.SceneUpdate entirely
	*pathtracer.SceneUpdate
}

type gameRendererImpl struct {
	transformT0 []geometry.TRS3

	// TODO: this should be a part of sceneUpdate. We also need the "currently
	// being rendered time interval" so that we can map input back to that.
	stuffMu        sync.Mutex
	t0sdl, t1sdl   uint64 // TODO: special type to represent SDL ticks?
	t0game, t1game game.Time
	// TODO: what if we want to pass multiple cameras to the composition
	// pipeline?
	ourCamera pathtracer.Camera

	// The update that didn't fit into the queue
	stagingUpdate *sceneUpdate
	// Queue of updates, consumed by Render
	updates chan *sceneUpdate

	// Only used by Render
	scene *pathtracer.Scene

	// Uhh
	sfxScene *sfx.Scene
}

// TODO: should be created as needed
var gameRenderer = &gameRendererImpl{
	transformT0: make([]geometry.TRS3, 10000),

	updates: make(chan *sceneUpdate, 1),

	scene: pathtracer.NewScene(10000, 5),

	sfxScene: &sfx.Scene{
		Instance: make([]sfx.Instance, 10000),
	},
}

// TODO: just inline Tick into this and kill this function once we can make
// surrounding code prettier
func (re *gameRendererImpl) updateInternal(f func(update *sceneUpdate)) {
	if re.stagingUpdate == nil {
		// TODO: pool this stuff
		re.stagingUpdate = &sceneUpdate{SceneUpdate: pathtracer.NewSceneDirty(10000)}
	}

	f(re.stagingUpdate)

	select {
	case re.updates <- re.stagingUpdate:
		re.stagingUpdate = nil
	default:
	}
}

func (re *gameRendererImpl) Tick(w *game.Scene, playerID ecs.ID, t0, t1 game.Time, frameDuration time.Duration) {
	conf := config.Load()

	re.updateInternal(func(update *sceneUpdate) {
		{
			for i := range update.Parent {
				update.Parent[i] = 0
			}

			for i := range update.Instance {
				update.Instance[i].Transform = 0
			}

			update.Sky = texture(w.Sky).Image

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

				transformT0 := re.transformT0[i]
				transformT1 := geometry.TRS3{
					Translation: tr.Translation.Add(geometry.DVec3FromVec3(offset)).Vec3(),
					Rotation:    tr.Rotation,
					Scale:       w.GetScale(id),
				}

				update.TransformT0[i] = transformT0
				update.TransformT1[i] = transformT1
				re.transformT0[i] = transformT1
			}

			// TODO: we need to split operations on caches into probe and fetch, so
			// that when a probe fails, we spawn a new goroutine with fetch and
			// everything that follows.

			for id, v := range ecs.Join(w.RenderingGeometry, w.TranslationRotation) {
				entity, hasEntity := w.Entity.Get(id)
				renderingGeometry := v.V1

				i := id.Index()

				mask := uint8(0b11)

				viewmodel, hasViewmodel := w.Viewmodel2.Get(id)
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

				// TODO: we'll be able to stop allocating a new slice every time
				// when we pool scene updates themselves
				update.Materials[i] = make([]pathtracer.MaterialInstance, len(mesh.defaultMaterials))

				for j := range update.Materials[i] {
					m2 := getmaterial(mesh.defaultMaterials[j])
					materialInstance := &update.Materials[i][j]
					materialInstance.Material = m2.material
					if hasEntity {
						m2.gatherArgs(materialInstance.Args[:], reflect.ValueOf(entity))
					}
				}

				update.Instance[i].Mask = mask
				update.Instance[i].Transform = i
			}

			// TODO: this should not exist and be part of sceneUpdate
			re.stuffMu.Lock()
			// TODO: make the caller responsible for passing wall clock time?
			re.t0sdl = sdl.TicksNS()
			re.t1sdl = re.t0sdl + uint64(frameDuration) // depends on timescale
			re.t0game = t0
			re.t1game = t1
			// TODO: factor out into a function, this gets reused in Subtick
			re.ourCamera = pathtracer.Camera{
				Transform:     update.Transform(fpsCharacter.Camera.Index(), 0),
				FieldOfView:   float32(geometry.Radians(67.5)),
				NearClipPlane: 0.01,
			}
			re.stuffMu.Unlock()
		}

		// Ughhhhhhh
		{
			playerEntity, _ := w.Entity.Get(playerID)
			fpsCharacter := playerEntity.(game.FPSCharacter)
			camera := update.Transform(fpsCharacter.Camera.Index(), 0)
			cameraPos := geometry.Vec3{camera[0][3], camera[1][3], camera[2][3]}

			scene := re.sfxScene

			a := int64(t1.Sub(t0) * 48000 / 1e9)

			clear(scene.Instance)

			for id, soundEffect := range w.SoundEffect.All() {
				positionRotation, _ := w.TranslationRotation.Get(id)
				scale, _ := w.Scale.Get(id)

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
			renderAudio(re.sfxScene, cameraPos, int64(t0.Sub(game.Time(0))*48000/1e9), a)
		}
	})
}

func (re *gameRendererImpl) Subtick(w *game.Scene, playerID ecs.ID) {
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

// TODO: should not be global lmao
var fn uint32

func (re *gameRendererImpl) Render(jq *gpu.JobQueue, dst *gpu.Image) {
	conf := config.Load()

	select {
	case update := <-re.updates:
		re.scene.EnqueueUpdate(jq, update.SceneUpdate, 0)
	default:
	}

	re.stuffMu.Lock()
	t := 1.0
	if true /* !conf.DontInterpolate */ {
		// TODO: we need to be able to lock sceneMu for this but we can't.
		// We should make our own scene type with the timestamps and stuff.
		t = min(max(float64(sdl.TicksNS()-re.t0sdl)/float64(re.t1sdl-re.t0sdl), 0), 1)
	}
	ourCamera := re.ourCamera
	re.stuffMu.Unlock()

	re.scene.Render(
		jq,
		pathtracer.Film{
			Extent: dst.Extent(),
			Color:  dst,
		},
		float32(t),
		fn,
		&ourCamera,
		&pathtracer.Quality{
			MaxBounces:               conf.Quality.MaxBounces,
			RussianRouletteThreshold: conf.Quality.RussianRouletteThreshold,
		})
	fn++
}
