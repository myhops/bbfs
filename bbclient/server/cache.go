package server

import (
	"sync"
	"time"

	"github.com/maypok86/otter"
)

type syncedCache[K comparable, V any] struct {
	cache otter.Cache[K, V]
	clearMutex sync.RWMutex
}

func NewCache[K comparable, V any]() *syncedCache[K, V] {
	c, err := otter.MustBuilder[K, V](10_000).
		CollectStats().
		Cost(func(key K, data V) uint32 {
			return 1
		}).
		WithTTL(time.Hour).
		Build()
	if err != nil {
		panic(err)
	}
	return &syncedCache[K, V]{
		cache: c,
	}
}

func (b *syncedCache[K, V]) Set(key K, value V) bool {
	b.clearMutex.RLock()
	defer b.clearMutex.RUnlock()
	return b.cache.Set(key, value)
}

func (b *syncedCache[K, V]) Get(key K) (V, bool) {
	b.clearMutex.RLock()
	defer b.clearMutex.RUnlock()
	return b.cache.Get(key)
}

func (b *syncedCache[K, V]) Clear() {
	b.clearMutex.Lock()
	defer b.clearMutex.Unlock()
	b.cache.Clear()
}
