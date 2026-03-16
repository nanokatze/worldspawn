package game

import (
	"sync"

	"github.com/go-json-experiment/json"

	"worldspawn/internal/gmath"
)

// TODO: move this stuff into its own package probably

// TODO: should not be defined here.
//
// TODO: when we move things elsewhere, we should make the internals private and
// introduce accessor methods instead.
//
// TODO: stick various methods onto the Pose object
type Pose struct {
	Bones map[string]gmath.Mat4x4
}

/*
type AnimationSampler struct {
	// repeat or clamp to edge
	AddressMode  int
	Unnormalized bool
}
*/

/*
type AnimationSamples struct {
}

type Animation2 struct {
	Channels map[string]AnimationSamples
}

type Animation struct {
	Armature map[string]gmath.TRS3 // TODO: split off into its own component?
	Action   string
	// Channels   map[string][]geometry.Affine3
	// SampleRate int // TODO: this must live as part of Channels. We separately should have a knob for animation playback speed or animation deadline.
	PlayTime Time

	// TODO: playback speed. We'll want both options: a speed factor as well as
	// deadline by which the animation must complete
}

// TODO: these should be getArmature and getAnimation the way we have getShape
// TODO: probably decorate errors with getArmature: $filename: $err

type Action struct {
	sampleRate int
	// TODO: interleave the channels
	// TODO: see what we can do about types of stuff we're interpolating
	samples map[string][]gmath.TRS3
}

// TODO: Sample should take channel *index* instead of name, and wrapping
// behavior
func (a *Action) Sample(t float64, channel string) gmath.TRS3 {
	samples := a.samples[channel]

	t *= float64(a.sampleRate)

	i := min(max(int(math.Floor(t)), 0), len(samples)-1)
	j := min(max(int(math.Ceil(t)), 0), len(samples)-1)

	return samples[i].NLerp(samples[j], float32(t-math.Floor(t)))
}

var actionCacheMu sync.Mutex
var actionCache = make(map[string]*Action)

func action(filename string) *Action {
	actionCacheMu.Lock()
	action, ok := actionCache[filename]
	actionCacheMu.Unlock()
	if !ok {
		f, err := Data.Open(filename)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		var m map[string][]gmath.TRS3
		if err := json.UnmarshalRead(f, &m, json.StringifyNumbers(true)); err != nil {
			panic(err)
		}

		// TODO: export sample rate to file so we can load it
		sampleRate := 60
		if filename == "anim_test.json" {
			sampleRate = 100
		}

		action = &Action{sampleRate: sampleRate, samples: m}

		actionCacheMu.Lock()
		actionCache[filename] = action
		actionCacheMu.Unlock()
	}
	return action
}
*/

type Skeleton struct {
	// Joints          []string

	// TODO: switch to a plain array with a string map for lookups

	Parent          map[string]string
	BindPose        map[string]gmath.Mat4x4
	BindPoseInverse map[string]gmath.Mat4x4

	// other stuff
}

var skeletonCache sync.Map

func skeleton(filename string) *Skeleton {
	if m, ok := skeletonCache.Load(filename); ok {
		return m.(*Skeleton)
	}

	m, err := loadSkeleton(filename)
	if err != nil {
		panic(err)
	}
	m2, _ := skeletonCache.LoadOrStore(filename, m)
	return m2.(*Skeleton)
}

func loadSkeleton(filename string) (*Skeleton, error) {
	f, err := Data.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var bindPoseInverse map[string]gmath.Mat4x4
	if err := json.UnmarshalRead(f, &bindPoseInverse, json.StringifyNumbers(true)); err != nil {
		return nil, err
	}
	bindPose := make(map[string]gmath.Mat4x4)
	for k, v := range bindPoseInverse {
		bindPose[k] = v.Inverse()
	}
	return &Skeleton{
		BindPose:        bindPose,
		BindPoseInverse: bindPoseInverse,
	}, nil
}
