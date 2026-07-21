package skeleton

import (
	"encoding/json/v2"
	"io"
	"maps"
	"slices"
	"unique"

	"worldspawn/internal/gmath"
)

// TODO: introduce skeleton builder to simplify loader code

type SkeletonBuilder struct {
}

// TODO: make the internals private?
type Skeleton struct {
	JointNames   []unique.Handle[string]
	JointByName_ map[unique.Handle[string]]int

	Parent         []int
	Children       [][]int
	BindPose       []gmath.Affine3f32
	BindPoseInv    []gmath.Affine3f32
	ParentRelative []gmath.Affine3f32
}

func (s *Skeleton) JointByName(name unique.Handle[string]) int {
	if i, ok := s.JointByName_[name]; ok {
		return i
	}
	return -1
}

// TODO: stick various methods onto the Pose object instead of exposing the
// internals. We want to be able to get relative to rest and absolute joint
// transforms.
// TODO: alternatively just kill this object?
type Pose struct {
	Bones map[int]gmath.Affine3f32
}

// TODO: have loader subpackages
// TODO: accept byte slice instead of a reader?
func Read(r io.Reader) (*Skeleton, error) {
	var tmp struct {
		Parent   map[string]string
		BindPose map[string]gmath.Mat4x4f32
	}
	if err := json.UnmarshalRead(r, &tmp, json.StringifyNumbers(true)); err != nil {
		return nil, err
	}

	bones := slices.Sorted(maps.Keys(tmp.BindPose))
	bonesInv := maps.Collect(func(yield func(string, int) bool) {
		for k, v := range bones {
			yield(v, k)
		}
	})

	var skeleton Skeleton

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

	skeleton.BindPoseInv = slices.Collect(func(yield func(gmath.Affine3f32) bool) {
		for _, bone := range bones {
			yield(gmath.Affine3FromMat4x4(tmp.BindPose[bone]).Inv())
		}
	})

	skeleton.ParentRelative = slices.Collect(func(yield func(gmath.Affine3f32) bool) {
		for bone := range bones {
			umm := gmath.Affine3One[float32]()
			if parent := skeleton.Parent[bone]; parent != -1 {
				umm = skeleton.BindPoseInv[parent]
			}
			yield(umm.Mul(skeleton.BindPose[bone]))
		}
	})

	return &skeleton, nil
}
