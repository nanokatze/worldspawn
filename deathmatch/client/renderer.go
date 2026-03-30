package main

import (
	"fmt"
	"io"
	"path"
	"reflect"
	"sync"
	"time"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/gpu"
	"worldspawn/internal/ecs"
	sfx "worldspawn/internal/fuckwwise"
	"worldspawn/internal/fuckwwise/opusfile"
	"worldspawn/internal/fuckwwise/wav"
	"worldspawn/internal/gmath"
	"worldspawn/internal/pathtracer"
	"worldspawn/internal/sdl"
)

type sceneUpdate struct {
	tm timeMapping

	camera          pathtracer.Camera
	cameraTransform int

	Sky *gpu.Image

	Parent      []int
	TransformT0 []gmath.TRS3f32
	TransformT1 []gmath.TRS3f32
	// TODO: we also need to carry velocity here for motion blur, or at least
	// some extra info to disambiguate fast temporally-aliased motions.

	Mask []uint8

	GeoNodes []geoNodes // TODO: rename this

	Materials    [][]*pathtracer.InterpretedMaterial
	MaterialArgs [][][256]byte
}

func newSceneDirty(n int) *sceneUpdate {
	return &sceneUpdate{
		Parent:      make([]int, n),
		TransformT0: make([]gmath.TRS3f32, n),
		TransformT1: make([]gmath.TRS3f32, n),

		Mask: make([]uint8, n),

		GeoNodes: make([]geoNodes, n),

		Materials:    make([][]*pathtracer.InterpretedMaterial, n),
		MaterialArgs: make([][][256]byte, n),
	}
}

// TODO: rename to something like GlobalTransform?
// TODO: this should use DAffine3
func (s *sceneUpdate) Transform(i int, t float32) gmath.Affine3f32 {
	B := gmath.Affine3One[float32]()
	for ; i != -1; i = s.Parent[i] {
		A := s.TransformT0[i].NLerp(s.TransformT1[i], t).Compose()
		B = A.Mul(B)
	}
	return B
}

type timeMapping struct {
	t0sdl, t1sdl   uint64 // TODO: special type to represent SDL ticks?
	t0game, t1game game.Time
}

type renderer struct {
	lastGen       []uint32
	lastTransform []gmath.TRS3f32

	// The update that didn't fit into the queue
	stagingUpdate *sceneUpdate
	// Queue of updates, consumed by Render
	updates chan *sceneUpdate

	frameNumber uint32
	tmMu        sync.Mutex
	tm          timeMapping
	// TODO: what if we want to pass multiple cameras to the composition
	// pipeline?
	// TODO: camera states need to be t0 and t1 too
	ourCamera          pathtracer.Camera
	ourCameraTransform int
	scene2             *sceneUpdate
	gsdata             []gsdata
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

	update := re.beginUpdate()
	defer re.commitUpdate(update)

	fpsCharacter, _ := game.SceneGetEntity[game.Player](w, playerID)

	camera := fpsCharacter.Camera(w)

	{
		for i := range update.Parent {
			update.Parent[i] = -1
		}

		update.Sky = texture(w.Globals().Sky).Image

		for id, tr := range ecs.All(&w.TransformTR) {
			i := id.Index()

			s, ok := w.TransformS.Get(id)
			if !ok {
				s = gmath.Mat3x3UOne[float32]()
			}

			cosmeticOffset, _ := w.CosmeticOffset.Get(id)

			var offset gmath.Vec3f32
			if !conf.Developer.DisableCosmeticOffset {
				offset = cosmeticOffset.Eval(w.Now)
			}

			trs := gmath.TRS3f32{
				// TODO: we should not record cosmetic offset into renderer.transformT0
				T: gmath.Vec3Convert[float32](tr.T).Add(offset),
				R: tr.R,
				S: s,
			}

			parent := w.GetParent(id)
			if parent != 0 {
				update.Parent[i] = parent.Index()
			}

			if parentBone, parentedToBone := w.ParentBone.Get(id); parentedToBone {
				pose, _ := w.Pose.Get(parent)
				parentBoneIndex := pose.Skelly.JointByName(parentBone)
				tmp := pose.Bones[parentBoneIndex].Mul(pose.Skelly.BindPose[parentBoneIndex])

				// kinda yikes but will do for now
				//
				// TODO: teach the renderer to understand skelly hierarchy and
				// thus properly interpolate the transform?
				trs = tmp.Mul(trs.Compose()).TRS()
			}

			transformT1 := trs

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

		for id, v := range ecs.Join(&w.RenderingGeometry, &w.TransformTR) {
			renderingGeometry := v.V1

			i := id.Index()

			visibility, _ := w.VisibilityMask.Get(id)
			mask := visibility.Mask & 0b11
			if visibility.Camera != camera {
				mask ^= 0b11
			}
			update.Mask[i] = mask

			geometry := getgeometry(renderingGeometry)

			pose, _ := w.Pose.Get(id)

			update.GeoNodes[i] = geoNodes{
				src:  geometry,
				pose: pose,
			}

			// TODO: stop allocating a new slice every time
			update.Materials[i] = make([]*pathtracer.InterpretedMaterial, len(geometry.materials))
			update.MaterialArgs[i] = make([][256]byte, len(geometry.materials))

			for j := range update.Materials[i] {
				m2 := getmaterial(geometry.materials[j])
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

		// TODO: also the camera itself might not be valid or w/e
		update.cameraTransform = camera.Index()
		update.camera = pathtracer.Camera{
			FieldOfView:   float32(gmath.Radians(67.5)),
			NearClipPlane: 0.01,
		}
	}

	// Ughhhhhhh
	{
		camera := update.Transform(camera.Index(), 0)
		cameraPos := camera.T

		scene := re.sfxScene

		a := int64(t1.Sub(t0) * 48000 / 1e9)

		clear(scene.Instance)

		for id, soundEffect := range ecs.All(&w.SoundEffect) {
			T, _ := w.GetGlobalTransform(id)

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
				Transform:   T,
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

// +Y forward +X right +Z up to -Z forward +X right -Y up
//
// TODO: move somewhere further up
// TODO: improve naming?
var fixup = gmath.Mat4x4f32{
	1, 0, 0, 0,
	0, 0, -1, 0,
	0, -1, 0, 0,
	0, 0, 0, 1,
}

func (re *renderer) Render(jq *gpu.JobQueue, sdlNow uint64, dst *gpu.Image) {
	conf := config.Load()

	select {
	case update := <-re.updates:
		re.tmMu.Lock()
		re.tm = update.tm
		re.tmMu.Unlock()
		re.ourCamera = update.camera
		re.ourCameraTransform = update.cameraTransform
		re.scene2 = update
		re.scene.SetSky(
			gmath.Mat3x3f32{
				0, -1, 0,
				0, 0, 1,
				1, 0, 0,
			},
			update.Sky)
		for i := range update.Mask {
			update.GeoNodes[i].EnqueueEvaluate(jq, &re.gsdata[i])

			geometry, accel := update.GeoNodes[i].Outputs(&re.gsdata[i])

			re.scene.SetInstanceGeometry(i, update.Mask[i], geometry, accel, update.Materials[i], update.MaterialArgs[i])
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
		camera.Transform = re.scene2.Transform(re.ourCameraTransform, float32(t)).ToMat().Mul(fixup.Inverse())

		for i := range re.scene2.Mask {
			tmp := re.scene2.Transform(i, float32(t))
			// TODO: outline this?
			var tmp2 [3][4]float32
			for i := range tmp2 {
				tmp2[i][0] = *tmp.M.Index(i, 0)
				tmp2[i][1] = *tmp.M.Index(i, 1)
				tmp2[i][2] = *tmp.M.Index(i, 2)
				tmp2[i][3] = tmp.T[i]
			}
			re.scene.SetInstanceTransform(i, tmp2)
		}
	}

	re.scene.EnqueueUpdateAccel(jq)

	re.scene.Render(
		jq,
		re.frameNumber,
		&camera,
		pathtracer.Film{
			Extent: [2]int(dst.Extent()),
			Color:  dst,
		},
		&pathtracer.Quality{
			MaxBounces:               conf.Quality.MaxBounces,
			RussianRouletteThreshold: conf.Quality.RussianRouletteThreshold,
		})
	re.frameNumber++
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
	if !v.IsValid() {
		return [4]float32{}
	}
	// TODO: allow types to implement a thing to get the value with
	if v.Type() == reflect.TypeFor[[4]float32]() {
		return v.Interface().([4]float32)
	}
	return [4]float32{}
}
