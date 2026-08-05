package main

import (
	"reflect"
	"sync"
	"time"

	"worldspawn/deathmatch/internal/game"
	"worldspawn/deathmatch/internal/hud"
	"worldspawn/gpu"
	"worldspawn/internal/ecs"
	"worldspawn/internal/gmath"
	"worldspawn/internal/renderer"
	"worldspawn/internal/sdl"
)

// TODO: rename the objects in here

// TODO: replace this with a widget tree (encoded sequentially like gio's ops,
// for efficiency)
type hudState struct {
	Health int32
	Bleed  int32
}

type gameRendererVideo struct {
	n int

	// TODO: instead of idGen, look at whether T_0 + velocity * dt is
	// too far from T_1
	idGen     []uint32
	transform []gmath.Affine3f32

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
	scene              *renderer.Scene
	hudState           hud.State
}

func (re *gameRendererVideo) Reset(n int) {
	// NOTE: this is called concurrently with Redraw. Keep that in mind when
	// implementing this function.
}

type sceneUpdate struct {
	tm timeMapping

	// TODO: this doesn't really need to be here, I think? We can just update it
	// directly (but it would have to be mutex protected.)
	hudState hud.State

	camera          renderer.Camera
	cameraTransform int

	sky *gpu.Image

	// TODO: make transformT0 and T1 actually just gmath.Affine3
	// parent      []int
	transformT0 []gmath.Affine3f32
	transformT1 []gmath.Affine3f32
	// TODO: we also need to carry velocity here for motion blur, or at least
	// some extra info to disambiguate fast temporally-aliased motions.

	mask []uint8

	geoNodes []geoNodes // TODO: rename this

	materials    [][]*renderer.InterpretedMaterial
	materialArgs [][][256]byte
}

func newSceneDirty(n int) *sceneUpdate {
	return &sceneUpdate{
		transformT0: make([]gmath.Affine3f32, n),
		transformT1: make([]gmath.Affine3f32, n),

		mask: make([]uint8, n),

		geoNodes: make([]geoNodes, n),

		materials:    make([][]*renderer.InterpretedMaterial, n),
		materialArgs: make([][][256]byte, n),
	}
}

func (s *sceneUpdate) Transform(i int, t float32) gmath.Affine3f32 {
	return s.transformT0[i].Scale(1 - t).Add(s.transformT1[i].Scale(t))
}

type timeMapping struct {
	t0sdl, t1sdl   uint64 // TODO: special type to represent SDL ticks?
	t0game, t1game game.Time
}

// TODO: remove this in favor of merging updates at commitUpdate time. I.e.
// we'll start off with a clean update every time.
func (re *gameRendererVideo) beginUpdate() *sceneUpdate {
	if re.stagingUpdate == nil {
		// TODO: pool this stuff
		return newSceneDirty(re.n)
	}
	tmp := re.stagingUpdate
	re.stagingUpdate = nil
	return tmp
}

// TODO: rename to enqueueUpdate?
func (re *gameRendererVideo) commitUpdate(update *sceneUpdate) {
	select {
	case re.updates <- update:
	default:
	}
}

func (re *gameRendererVideo) Update(world *game.World, playerID ecs.ID, t0, t1 game.Time, frameDuration time.Duration) {
	// TODO: pass the bits that interest us explicitly
	//conf := config.Load()

	// TODO: move most of this code into game. We should introduce a
	// World.Render method which could either return a pile of bytes, or be fed
	// an interface.

	update := re.beginUpdate()
	defer re.commitUpdate(update)

	camera := world.Camera(playerID)

	{
		update.hudState.Update(world, playerID)

		update.sky = texturecache.Get(world.Globals().Sky).Image

		for id := range ecs.All(&world.TransformTR) {
			i := id.Index()

			transformT1 := world.GetRenderingTransform(world.Entity(id)).Convert[float32]()

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
			entity := world.Entity(id)

			renderingGeometry := v.V1

			i := id.Index()

			visibility, _ := world.VisibilityCondition.Get(id)
			mask := visibility.Mask & 0b11
			if visibility.Camera != camera {
				mask ^= 0b11
			}
			update.mask[i] = mask

			geometry := modelcache.Get(renderingGeometry)

			pose := entity.Pose()

			update.geoNodes[i] = geoNodes{
				src:    geometry,
				skelly: world.GetSkeleton(id),
				pose:   pose,
			}

			// TODO: stop allocating a new slice every time
			update.materials[i] = make([]*renderer.InterpretedMaterial, len(geometry.materials))
			update.materialArgs[i] = make([][256]byte, len(geometry.materials))

			for j := range update.materials[i] {
				material := materialcache.Get(geometry.materials[j])
				update.materials[i][j] = material.material
				material.preamble.Pack(update.materialArgs[i][j][:],
					func(name string, out *[4]float32) {
						state := world.Entity(id).ScriptState()
						if state != nil {
							*out = getv(reflect.ValueOf(state).FieldByName(name))
						}
					})
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
}

func (re *gameRendererVideo) UpdateSubtick(world *game.World, playerID ecs.ID) {
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

func (re *gameRendererVideo) Redraw(jq *gpu.JobQueue, dst *gpu.Image, sdlNow uint64) {
	conf := config.Load()

	select {
	case update := <-re.updates:
		re.tmMu.Lock()
		re.tm = update.tm
		re.tmMu.Unlock()
		re.hudState = update.hudState
		re.ourCamera = update.camera
		re.ourCameraTransform = update.cameraTransform
		re.scene2 = update
		re.scene.SetSky(
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

			re.scene.SetInstanceGeometry(i, update.mask[i], geometry, accel, update.materials[i], update.materialArgs[i])
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
		cameraTransform = re.scene2.Transform(re.ourCameraTransform, float32(t)).Mat().Mul(worldspawnToRenderer.Inv())

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
			re.scene.SetInstanceTransform(i, tmp2)
		}
	}

	re.scene.EnqueueUpdateAccel(jq)
	re.scene.EnqueueRender(
		jq,
		renderer.Film{
			Extent: [2]int(dst.Extent()),
			Color:  dst,
		},
		&camera,
		cameraTransform,
		re.frameNumber,
		&renderer.Quality{
			MaxBounces:               conf.Quality.MaxBounces,
			RussianRouletteThreshold: conf.Quality.RussianRouletteThreshold,
		})

	re.hudState.Draw(jq, dst)

	re.frameNumber++
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
