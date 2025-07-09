package parallel

import (
	"iter"
	"runtime"
	"sync"
)

// TODO: also entertain the idea of somehow hooking things up for Jolt. Jolt
// wants a task graph kind of thing to work with. This thing only deals with
// independent loops. I guess we should at least keep this thing because it's
// pretty convenient.

// TODO: tidy this up

type goroutinePool struct {
	queues   []chan func() // TODO: these things should be created lazily, but we'll always have runtime.NumCPU() of these
	rotation int           // next queue to push stuff into
	wg       sync.WaitGroup
}

func newGoroutinePool() *goroutinePool {
	gpool := new(goroutinePool)
	gpool.queues = make([]chan func(), runtime.NumCPU())
	return gpool
}

func (gpool *goroutinePool) Go(job func()) {
	if gpool.queues[gpool.rotation] == nil {
		// TODO: outline this into a function
		jq := make(chan func(), 1) // TODO: this should be tunable at compile time. For good perf, it should be at least 1
		gpool.queues[gpool.rotation] = jq
		gpool.wg.Add(1)
		go func() {
			// TODO: could we avoid terminating these goroutines until we get
			// collected by GC? Would that be worth it?

			defer gpool.wg.Done()

			for {
				job := <-jq
				if job == nil {
					return
				}
				job()
			}
		}()
	}
	gpool.queues[gpool.rotation] <- job
	gpool.rotation += 1
	if gpool.rotation >= len(gpool.queues) {
		gpool.rotation = 0
	}
}

func (gpool *goroutinePool) Finish() {
	for _, q := range gpool.queues {
		q <- nil
	}
	gpool.wg.Wait()
}

// TODO: provide specialization of this for slices and or maps? Idk
func For[K, V any](seq iter.Seq2[K, V], chunkSize int, f func(k K, v V)) {
	if false {
		for k, v := range seq {
			f(k, v)
		}
		return
	}

	gpool := newGoroutinePool()
	defer gpool.Finish()

	// TODO: make it type chunk[K, V]
	type Chunk struct {
		Ks []K
		Vs []V
	}

	// TODO: have a sync.Pool of Chunks?

	// TODO: should be job *Chunk, i, j int
	makeJob := func(job Chunk) func() {
		return func() {
			_ = job.Ks[len(job.Vs)-1]
			_ = job.Vs[len(job.Ks)-1]
			for i := range job.Ks {
				k := job.Ks[i]
				v := job.Vs[i]
				f(k, v)
			}
		}
	}

	var chunk Chunk
	for k, v := range seq {
		if len(chunk.Ks) == cap(chunk.Ks) {
			// TODO: move this into a separate function and keep it outlined. We
			// could also perhaps move this to happen after batch append.
			func() {
				if len(chunk.Ks) > 0 {
					gpool.Go(makeJob(chunk))
				}
				chunk.Ks = make([]K, 0, chunkSize)
				chunk.Vs = make([]V, 0, chunkSize)
			}()
		}

		chunk.Ks = append(chunk.Ks, k)
		chunk.Vs = append(chunk.Vs, v)
	}
	// TODO: we should split off the chunks if we didn't get even a single one
	// to run, I think? Basically just do len(chunk)/len(workers)
	// subdivisions... If we're going to pass *Chunk around, we will need to
	// pass start and end offsets explicitly I think.
	if len(chunk.Ks) > 0 {
		gpool.Go(makeJob(chunk))
	}
}
