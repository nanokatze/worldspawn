package game

import (
	"sync"
	"unique"

	"worldspawn/internal/animation"
	"worldspawn/internal/loaders/skeleton"
)

// TODO: move this stuff into its own package probably

var animationCache sync.Map

func getanimation(filename string) *animation.Animation {
	if m, ok := animationCache.Load(filename); ok {
		return m.(*animation.Animation)
	}

	m, err := loadAnimation(filename)
	if err != nil {
		panic(err)
	}
	m2, _ := animationCache.LoadOrStore(filename, m)
	return m2.(*animation.Animation)
}

func loadAnimation(filename string) (*animation.Animation, error) {
	f, err := Data.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return animation.Read(f)
}

var skeletonCache sync.Map

func getskeleton(filename unique.Handle[string]) *skeleton.Skeleton {
	if m, ok := skeletonCache.Load(filename); ok {
		return m.(*skeleton.Skeleton)
	}

	m, err := loadSkeleton(filename.Value())
	if err != nil {
		panic(err)
	}
	m2, _ := skeletonCache.LoadOrStore(filename, m)
	return m2.(*skeleton.Skeleton)
}

func loadSkeleton(filename string) (*skeleton.Skeleton, error) {
	f, err := Data.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return skeleton.Read(f)
}
