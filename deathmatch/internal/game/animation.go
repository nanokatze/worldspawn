package game

import (
	"unique"

	"worldspawn/internal/animation"
	"worldspawn/internal/cache"
	"worldspawn/internal/loaders/skeleton"
)

var animationCache = cache.New(func(key unique.Handle[string]) *animation.Animation {
	f, err := Data.Open(key.Value())
	if err != nil {
		panic(err)
	}
	defer f.Close()

	animation, err := animation.Read(f)
	if err != nil {
		panic(err)
	}

	return animation
})

var skeletonCache = cache.New(func(key unique.Handle[string]) *skeleton.Skeleton {
	f, err := Data.Open(key.Value())
	if err != nil {
		panic(err)
	}
	defer f.Close()

	skeleton, err := skeleton.Read(f)
	if err != nil {
		panic(err)
	}

	return skeleton
})
