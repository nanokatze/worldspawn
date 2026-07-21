package skeleton

import (
	"encoding/json/v2"
	"io"
	"unique"

	"worldspawn/internal/gmath"
)

type Skeleton struct {
	JointNames   []unique.Handle[string]
	JointByName_ map[unique.Handle[string]]int

	Parent   []int
	Children [][]int

	BindPose    []gmath.Affine3f32
	BindPoseInv []gmath.Affine3f32

	ParentRelative []gmath.Affine3f32
}

func (s *Skeleton) NumJoints() int {
	return len(s.JointNames)
}

func (s *Skeleton) JointByName(name unique.Handle[string]) int {
	if i, ok := s.JointByName_[name]; ok {
		return i
	}
	return -1
}

// TODO: move actual loaders into subpackages
func Read(r io.Reader) (*Skeleton, error) {
	type joint struct {
		Name     string
		Parent   int
		BindPose gmath.Mat4x4f32
	}
	var joints []joint
	if err := json.UnmarshalRead(r, &joints, json.StringifyNumbers(true)); err != nil {
		return nil, err
	}

	var skeleton Skeleton
	skeleton.JointNames = make([]unique.Handle[string], len(joints))
	skeleton.JointByName_ = make(map[unique.Handle[string]]int, len(joints))
	skeleton.Parent = make([]int, len(joints))
	skeleton.Children = make([][]int, len(joints))
	skeleton.BindPose = make([]gmath.Affine3f32, len(joints))
	skeleton.BindPoseInv = make([]gmath.Affine3f32, len(joints))
	skeleton.ParentRelative = make([]gmath.Affine3f32, len(joints))

	for i, joint := range joints {
		name := unique.Make(joint.Name)
		skeleton.JointNames[i] = name
		skeleton.JointByName_[name] = i

		bindPose := gmath.Affine3FromMat4x4(joint.BindPose)
		skeleton.BindPose[i] = bindPose
		skeleton.BindPoseInv[i] = bindPose.Inv()
	}

	// TODO: require that on disk children appear strictly after their parents.
	// This will permit us to load everything in a single pass.

	for i, joint := range joints {
		parent := joint.Parent
		skeleton.Parent[i] = parent
		parentBindPoseInv := gmath.Affine3One[float32]()
		if parent != -1 {
			skeleton.Children[parent] = append(skeleton.Children[parent], i)
			parentBindPoseInv = skeleton.BindPoseInv[parent]
		}
		skeleton.ParentRelative[i] = parentBindPoseInv.Mul(skeleton.BindPose[i])
	}

	return &skeleton, nil
}
