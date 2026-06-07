package main

import (
	"reflect"
	"sync"
	"time"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/gpu"
	"worldspawn/internal/arenderer"
	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
	"worldspawn/internal/renderer"
	"worldspawn/internal/sdl"
)

type sceneUpdate struct {
	tm timeMapping

	camera          renderer.Camera
	cameraTransform int

	sky *gpu.Image

	parent      []int
	transformT0 []gmath.TRS3f32
	transformT1 []gmath.TRS3f32
	// TODO: we also need to carry velocity here for motion blur, or at least
	// some extra info to disambiguate fast temporally-aliased motions.

	mask []uint8

	geoNodes []geoNodes // TODO: rename this

	materials    [][]*renderer.InterpretedMaterial
	materialArgs [][][256]byte
}

func newSceneDirty(n int) *sceneUpdate {
	return &sceneUpdate{
		parent:      make([]int, n),
		transformT0: make([]gmath.TRS3f32, n),
		transformT1: make([]gmath.TRS3f32, n),

		mask: make([]uint8, n),

		geoNodes: make([]geoNodes, n),

		materials:    make([][]*renderer.InterpretedMaterial, n),
		materialArgs: make([][][256]byte, n),
	}
}

// TODO: this should use Affine3f64
func (s *sceneUpdate) Transform(i int, t float32) gmath.Affine3f32 {
	A := gmath.Affine3One[float32]()
	for ; i != -1; i = s.parent[i] {
		T := s.transformT0[i].NLerp(s.transformT1[i], t).Compose()
		A = T.Mul(A)
	}
	return A
}

type timeMapping struct {
	t0sdl, t1sdl   uint64 // TODO: special type to represent SDL ticks?
	t0game, t1game game.Time
}

type rendererGlue struct {
	n int

	idGen []uint32
	// parent []int
	transform []gmath.TRS3f32

	// TODO: outline things into video and audio parts?

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
	ourCamera          renderer.Camera
	ourCameraTransform int
	scene2             *sceneUpdate
	gsdata             []gsdata
	gscene             *renderer.Scene

	// NOTE: sound renderer works very differently from graphics renderer: we'll
	// probably tie it to simulation ticks. Or I guess we could also piggyback
	// off renderer but that's varying degrees of annoying...
	//
	// Anyway, we'll have an equivalent of pathtracer.Scene but for audio.
	// EXCEPT, we'll also need to access the output buffers on host (I'm not
	// sure if these will be per-instance or per-point-on-a-sphere.)
	//
	// We'll also need to have a huge output buffer so that we can write sound
	// into the future (for doppler) and also a per-instance buffer of samples
	// that we'll use wavegen or whatever to write into.
	ascene *arenderer.Scene
}

func (re *rendererGlue) Reset(n int) {
	// NOTE: this is called concurrently with Redraw. Keep that in mind when
	// implementing this function.
}

// TODO: remove this in favor of merging updates at commitUpdate time. I.e.
// we'll start off with a clean update every time.
func (re *rendererGlue) beginUpdate() *sceneUpdate {
	if re.stagingUpdate == nil {
		// TODO: pool this stuff
		return newSceneDirty(re.n)
	}
	tmp := re.stagingUpdate
	re.stagingUpdate = nil
	return tmp
}

// TODO: rename to enqueueUpdate?
func (re *rendererGlue) commitUpdate(update *sceneUpdate) {
	select {
	case re.updates <- update:
	default:
	}
}

func (re *rendererGlue) Tick(world *game.World, playerID ecs.ID, t0, t1 game.Time, frameDuration time.Duration) {
	conf := config.Load()

	update := re.beginUpdate()
	defer re.commitUpdate(update)

	fpsCharacter, _ := game.SceneGetEntity[game.Player](world, playerID)

	camera := fpsCharacter.Camera(world)

	{
		for i := range update.parent {
			update.parent[i] = -1
		}

		update.sky = texture(world.Globals().Sky).Image

		for id, tr := range ecs.All(&world.TransformTR) {
			i := id.Index()

			s, ok := world.TransformS.Get(id)
			if !ok {
				s = gmath.Mat3x3UOne[float32]()
			}

			cosmeticOffset, _ := world.CosmeticOffset.Get(id)

			var offset gmath.Vec3f32
			if !conf.Developer.DisableCosmeticOffset {
				offset = cosmeticOffset.Eval(world.Now)
			}

			trs := gmath.TRS3f32{
				T: gmath.Vec3Convert[float32](tr.T).Add(offset),
				R: tr.R,
				S: s,
			}

			parent := world.GetParent(id)
			if parent != 0 {
				update.parent[i] = parent.Index()
			}

			if parentBone, parentedToBone := world.ParentBone.Get(id); parentedToBone {
				pose, _ := world.Pose.Get(parent)
				skelly := world.GetSkeleton(parent)
				parentBoneIndex := skelly.JointByName(parentBone)
				hmm, ok := pose.Bones[parentBoneIndex]
				if !ok {
					hmm = gmath.Affine3One[float32]()
				}
				tmp := hmm.Mul(skelly.BindPose[parentBoneIndex])

				// kinda yikes but will do for now
				//
				// TODO: teach the renderer to understand skelly hierarchy and
				// thus properly interpolate the transform?
				trs = tmp.Mul(trs.Compose()).TRS()
			}

			transformT1 := trs

			transformT0 := re.transform[i]
			if re.idGen[i] != id.Generation() {
				transformT0 = transformT1
			}

			update.transformT0[i] = transformT0
			update.transformT1[i] = transformT1

			re.idGen[i] = id.Generation()
			re.transform[i] = transformT1
		}

		// TODO: we need to split operations on caches into probe and fetch, so
		// that when a probe fails, we spawn a new goroutine with fetch and
		// everything that follows.

		for id, v := range ecs.Join(&world.RenderingGeometry, &world.TransformTR) {
			renderingGeometry := v.V1

			i := id.Index()

			visibility, _ := world.VisibilityMask.Get(id)
			mask := visibility.Mask & 0b11
			if visibility.Camera != camera {
				mask ^= 0b11
			}
			update.mask[i] = mask

			geometry := getgeometry(renderingGeometry)

			pose, _ := world.Pose.Get(id)

			update.geoNodes[i] = geoNodes{
				src:    geometry,
				skelly: world.GetSkeleton(id),
				pose:   pose,
			}

			// TODO: stop allocating a new slice every time
			update.materials[i] = make([]*renderer.InterpretedMaterial, len(geometry.materials))
			update.materialArgs[i] = make([][256]byte, len(geometry.materials))

			for j := range update.materials[i] {
				material := getmaterial(geometry.materials[j])
				update.materials[i][j] = material.material
				material.preamble.Call(update.materialArgs[i][j][:], &attributeGetter{world, id})
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
		update.camera = renderer.Camera{
			FieldOfView:   float32(gmath.Radians(67.5)),
			NearClipPlane: 0.01,
		}
	}

	{
		cameraTransform := update.Transform(camera.Index(), 0)

		scene := re.ascene

		clear(scene.Emitters)

		for id, soundEffect := range ecs.All(&world.SoundEffect) {
			T := world.GetGlobalTransform(id)

			effect := lookupsound(soundEffect.Effect)

			scene.Transform[id.Index()] = gmath.Affine3Convert[float32](T).TRS()

			hmm := min(max(int64(t0.Sub(soundEffect.PlayTime)*48000/1e9), 0), int64(len(effect)))
			scene.Emitters[id.Index()] = effect[hmm:]
		}

		// TODO: for long ticks we'd like to do several short audio renders per
		// tick
		renderAudio(re.ascene, cameraTransform, int64(t1.Sub(t0)*48000/1e9))
	}
}

func (re *rendererGlue) Subtick(world *game.World, playerID ecs.ID) {
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

var worldspawnToRenderer = gmath.Mat4x4f32{
	1, 0, 0, 0,
	0, 0, -1, 0,
	0, -1, 0, 0,
	0, 0, 0, 1,
}

func (re *rendererGlue) Redraw(jq *gpu.JobQueue, dst *gpu.Image, sdlNow uint64) {
	conf := config.Load()

	select {
	case update := <-re.updates:
		re.tmMu.Lock()
		re.tm = update.tm
		re.tmMu.Unlock()
		re.ourCamera = update.camera
		re.ourCameraTransform = update.cameraTransform
		re.scene2 = update
		re.gscene.SetSky(
			gmath.Mat3x3f32{
				0, -1, 0,
				0, 0, 1,
				1, 0, 0,
			},
			update.sky)
		for i := range re.n {
			// TODO: evaluate geometry nodes in parallel
			update.geoNodes[i].EnqueueEvaluate(jq, &re.gsdata[i])

			geometry, accel := update.geoNodes[i].Outputs(&re.gsdata[i])

			re.gscene.SetInstanceGeometry(i, update.mask[i], geometry, accel, update.materials[i], update.materialArgs[i])
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
	cameraTransform := gmath.Mat4x4One[float32]()
	if re.scene2 != nil {
		cameraTransform = re.scene2.Transform(re.ourCameraTransform, float32(t)).ToMat().Mul(worldspawnToRenderer.Inverse())

		for i := range re.scene2.mask {
			tmp := re.scene2.Transform(i, float32(t))
			// TODO: outline this?
			var tmp2 [3][4]float32
			for i := range tmp2 {
				tmp2[i][0] = *tmp.M.Index(i, 0)
				tmp2[i][1] = *tmp.M.Index(i, 1)
				tmp2[i][2] = *tmp.M.Index(i, 2)
				tmp2[i][3] = tmp.T[i]
			}
			re.gscene.SetInstanceTransform(i, tmp2)
		}
	}

	re.gscene.EnqueueUpdateAccel(jq)

	film := renderer.Film{
		Extent: [2]int(dst.Extent()),
		Color:  dst,
	}

	quality := renderer.Quality{
		MaxBounces:               conf.Quality.MaxBounces,
		RussianRouletteThreshold: conf.Quality.RussianRouletteThreshold,
	}

	re.gscene.EnqueueRender(jq, film, &camera, cameraTransform, re.frameNumber, &quality)
	re.frameNumber++
}

type attributeGetter struct {
	world  *game.World
	entity ecs.ID
}

func (getter *attributeGetter) UniformAttribute(name string, out *[4]float32) bool {
	objectData, ok := getter.world.Entity.Get(getter.entity)
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
