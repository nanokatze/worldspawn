package gpu

import "sync"

type notifyList struct {
	goroutines sync.Cond
	jobBatches []*jobBatch
}

func (l *notifyList) init(locker sync.Locker) {
	l.goroutines.L = locker
}

func (l *notifyList) addJobBatch(b *jobBatch) {
	l.jobBatches = append(l.jobBatches, b)
}

func (l *notifyList) notify() {
	// Wake up host goroutines
	l.goroutines.Broadcast()

	// Wake up job batches
	for _, b := range l.jobBatches {
		b.lock()
		b.deps--
		b.unlockAndMaybeMakeRunnable()
	}
	l.jobBatches = nil
}
