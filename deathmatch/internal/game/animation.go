package game

import (
	"sync"

	"github.com/go-json-experiment/json"

	"worldspawn/internal/animgraph"
	"worldspawn/internal/gmath"
)

// TODO: move this stuff into its own package probably

// TODO: kill
type Pose = animgraph.Pose

/*
func cached[T, U any](f func(x T) U) func(x T) U {
	var cache sync.Map

	return func(x T) U {
		if m, ok := cache.Load(x); ok {
			return m.(U)
		}

		m, _ := animationCache.LoadOrStore(x, f(x))
		return m.(U)
	}
}
*/

var animationCache sync.Map

func animation(filename string) *animgraph.Animation {
	if m, ok := animationCache.Load(filename); ok {
		return m.(*animgraph.Animation)
	}

	m, err := loadAnimation(filename)
	if err != nil {
		panic(err)
	}
	m2, _ := animationCache.LoadOrStore(filename, m)
	return m2.(*animgraph.Animation)
}

func loadAnimation(filename string) (*animgraph.Animation, error) {
	f, err := Data.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var animation animgraph.Animation
	if err := json.UnmarshalRead(f, &animation, json.StringifyNumbers(true)); err != nil {
		return nil, err
	}
	return &animation, nil
}

type Skeleton struct {
	// Joints          []string

	// TODO: switch to a plain array with a string map for lookups

	Parent          map[string]string
	BindPose        map[string]gmath.Affine3f32
	BindPoseInverse map[string]gmath.Affine3f32
	ParentRelative  map[string]gmath.Affine3f32

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

	var tmp struct {
		Parent   map[string]string
		BindPose map[string]gmath.Mat4x4f32
	}
	if err := json.UnmarshalRead(f, &tmp, json.StringifyNumbers(true)); err != nil {
		return nil, err
	}

	var skeleton Skeleton

	skeleton.Parent = tmp.Parent

	skeleton.BindPose = make(map[string]gmath.Affine3f32)
	for bone, m := range tmp.BindPose {
		skeleton.BindPose[bone] = gmath.Affine3FromMat4x4(m)
	}

	skeleton.BindPoseInverse = make(map[string]gmath.Affine3f32)
	for bone, v := range skeleton.BindPose {
		skeleton.BindPoseInverse[bone] = v.Inv()
	}

	skeleton.ParentRelative = make(map[string]gmath.Affine3f32)
	for bone := range skeleton.BindPose {
		umm := gmath.Affine3One[float32]()
		if parent, hasParent := skeleton.Parent[bone]; hasParent {
			umm = skeleton.BindPoseInverse[parent]
		}
		skeleton.ParentRelative[bone] = umm.Mul(skeleton.BindPose[bone])
	}

	return &skeleton, nil
}
