package skeleton

import (
	"encoding/json/v2"
	"io"
	"unique"

	"worldspawn/internal/gmath"
)

// TODO: joint flags to control scaling inheritance etc during forward kinematics?

// TODO: make this an interface?
type Skeleton struct {
	JointNames   []unique.Handle[string]
	JointByName_ map[unique.Handle[string]]int

	Parent   []int
	Children [][]int

	BindPose    []gmath.Affine3f32
	BindPoseInv []gmath.Affine3f32

	ParentRelative []gmath.Affine3f32
}

func (sk *Skeleton) NumJoints() int {
	return len(sk.JointNames)
}

func (sk *Skeleton) JointByName(name unique.Handle[string]) int {
	if i, ok := sk.JointByName_[name]; ok {
		return i
	}
	return -1
}

func (sk *Skeleton) ForwardKinematics(a []gmath.Affine3f32, pose Pose) {
	// TODO: flooding would be more efficient but we need joints to be sorted
	// such that parents appear before children for flooding to not require
	// recursion.
	for bone := range sk.NumJoints() {
		A := gmath.Affine3One[float32]()

		tmp := bone
		for {
			A = sk.ParentRelative[tmp].Mul(a[tmp]).Mul(A)

			parent := sk.Parent[tmp]
			if parent == -1 {
				break
			}
			tmp = parent
		}

		pose[bone] = A.Mul(sk.BindPoseInv[bone])
	}
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

	var sk Skeleton
	sk.JointNames = make([]unique.Handle[string], len(joints))
	sk.JointByName_ = make(map[unique.Handle[string]]int, len(joints))
	sk.Parent = make([]int, len(joints))
	sk.Children = make([][]int, len(joints))
	sk.BindPose = make([]gmath.Affine3f32, len(joints))
	sk.BindPoseInv = make([]gmath.Affine3f32, len(joints))
	sk.ParentRelative = make([]gmath.Affine3f32, len(joints))

	for i, joint := range joints {
		name := unique.Make(joint.Name)
		sk.JointNames[i] = name
		sk.JointByName_[name] = i

		bindPose := gmath.Affine3FromMat(joint.BindPose)
		sk.BindPose[i] = bindPose
		sk.BindPoseInv[i] = bindPose.Inv()
	}

	// TODO: require that on disk children appear strictly after their parents.
	// This will permit us to load everything in a single pass.

	for i, joint := range joints {
		parent := joint.Parent
		sk.Parent[i] = parent
		parentBindPoseInv := gmath.Affine3One[float32]()
		if parent != -1 {
			sk.Children[parent] = append(sk.Children[parent], i)
			parentBindPoseInv = sk.BindPoseInv[parent]
		}
		sk.ParentRelative[i] = parentBindPoseInv.Mul(sk.BindPose[i])
	}

	return &sk, nil
}
