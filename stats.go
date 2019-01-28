package redirect

import (
	"sync"
	"sync/atomic"
)

type inMemoryStat struct {
	cache map[string]*int64
	lock  sync.RWMutex
}

func InMemoryStats() Stats {
	return &inMemoryStat{
		cache: make(map[string]*int64),
	}
}

func (ms *inMemoryStat) Touch(url string) {
	ms.lock.RLock()
	val, ok := ms.cache[url]
	ms.lock.RUnlock()
	if !ok {
		ms.lock.Lock()
		val, ok = ms.cache[url]
		if !ok {
			val = new(int64)
			ms.cache[url] = val
		}
		ms.lock.Unlock()
	}
	atomic.AddInt64(val, 1)
}

func (ms *inMemoryStat) Visits(url string) int64 {
	ms.lock.RLock()
	val, ok := ms.cache[url]
	ms.lock.RUnlock()
	if !ok {
		return 0
	}
	return *val
}
