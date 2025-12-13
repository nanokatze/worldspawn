package pathtracer

import (
	"worldspawn/geometry-go"
	"worldspawn/gpu"
)

// TODO: merge this file into scene

type Instance struct {
	Transform int

	Mask uint8
}

// TODO: we'll also want to specify the parameters
// TODO: move Material into its own column (array)?
type MaterialInstance struct {
	Material *InterpretedMaterial
	Args     [256]byte
}

// TODO: actually remove this entirely from here and push any kind of tracking
// onto the user.
type SceneUpdate struct {
	Sky *gpu.Image

	Parent      []int
	TransformT0 []geometry.TRS3
	TransformT1 []geometry.TRS3
	// TODO: we also need to carry velocity here for motion blur, or at least
	// some extra info to disambiguate fast temporally-aliased motions.

	// TODO: prefix instance-rate stuff with instance? or idk Or put them into
	// Instance struct { ... }
	Instance  []Instance
	Mesh      []*Mesh
	Materials [][]MaterialInstance
}

func NewSceneDirty(n int) *SceneUpdate {
	return &SceneUpdate{
		Parent:      make([]int, n),
		TransformT0: make([]geometry.TRS3, n),
		TransformT1: make([]geometry.TRS3, n),

		Instance:  make([]Instance, n),
		Mesh:      make([]*Mesh, n),
		Materials: make([][]MaterialInstance, n),
	}
}

// TODO: rename to something like GlobalTransform?
func (s *SceneUpdate) Transform(i int, t float32) geometry.Mat4x4 {
	B := geometry.Mat4x4Identity()
	for ; i != 0; i = s.Parent[i] {
		A := s.TransformT0[i].NLerp(s.TransformT1[i], t).Mat4x4()
		B = A.Mul4x4(B)
	}
	return B
}
