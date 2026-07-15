package game

import (
	"maps"
	"slices"
	"sync"
	"unique"

	"github.com/go-json-experiment/json"

	"worldspawn/internal/animgraph"
	"worldspawn/internal/gmath"
)

// TODO: move this stuff into its own package probably

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

var skeletonCache sync.Map

func skeleton(filename unique.Handle[string]) *animgraph.Skeleton {
	if m, ok := skeletonCache.Load(filename); ok {
		return m.(*animgraph.Skeleton)
	}

	m, err := loadSkeleton(filename.Value())
	if err != nil {
		panic(err)
	}
	m2, _ := skeletonCache.LoadOrStore(filename, m)
	return m2.(*animgraph.Skeleton)
}

func loadSkeleton(filename string) (*animgraph.Skeleton, error) {
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

	bones := slices.Sorted(maps.Keys(tmp.BindPose))
	bonesInv := maps.Collect(func(yield func(string, int) bool) {
		for k, v := range bones {
			yield(v, k)
		}
	})

	var skeleton animgraph.Skeleton

	skeleton.JointNames = slices.Collect(func(yield func(unique.Handle[string]) bool) {
		for _, bone := range bones {
			yield(unique.Make(bone))
		}
	})
	skeleton.JointByName_ = maps.Collect(func(yield func(unique.Handle[string], int) bool) {
		for bone, index := range bonesInv {
			yield(unique.Make(bone), index)
		}
	})

	skeleton.Parent = slices.Collect(func(yield func(int) bool) {
		for _, bone := range bones {
			parent, hasParent := skeleton.JointByName_[unique.Make(tmp.Parent[bone])]
			if !hasParent {
				parent = -1
			}
			yield(parent)
		}
	})

	skeleton.Children = make([][]int, len(skeleton.Parent))
	for bone, parent := range skeleton.Parent {
		if parent != -1 {
			skeleton.Children[parent] = append(skeleton.Children[parent], bone)
		}
	}

	skeleton.BindPose = slices.Collect(func(yield func(gmath.Affine3f32) bool) {
		for _, bone := range bones {
			yield(gmath.Affine3FromMat4x4(tmp.BindPose[bone]))
		}
	})

	skeleton.BindPoseInverse = slices.Collect(func(yield func(gmath.Affine3f32) bool) {
		for _, bone := range bones {
			yield(gmath.Affine3FromMat4x4(tmp.BindPose[bone]).Inv())
		}
	})

	skeleton.ParentRelative = slices.Collect(func(yield func(gmath.Affine3f32) bool) {
		for bone := range bones {
			umm := gmath.Affine3One[float32]()
			if parent := skeleton.Parent[bone]; parent != -1 {
				umm = skeleton.BindPoseInverse[parent]
			}
			yield(umm.Mul(skeleton.BindPose[bone]))
		}
	})

	return &skeleton, nil
}
