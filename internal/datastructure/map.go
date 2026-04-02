package datastructure

import "sync"

// SyncMap: Safe for concurrent use.
type SyncMap[K comparable, V any] struct {
	m sync.Map 
}

func (sm *SyncMap[K, V]) Store(key K, value *V) {
	sm.m.Store(key, value)
}

func (sm *SyncMap[K, V]) Load(key K) (*V, bool) {
	if val, ok := sm.m.Load(key); ok {
		return val.(*V), true
	}
	return nil, false
}

func (sm *SyncMap[K, V]) Update(key K, fn func(old *V) *V) {
	if actual, ok := sm.m.Load(key); ok {
		newValue := fn(actual.(*V))
		sm.m.Store(key, newValue)
		return
	}
	newValue := fn(nil)
	sm.m.Store(key, newValue)
}

func (sm *SyncMap[K, V]) Delete(key K) {
	sm.m.Delete(key)
}

func (sm *SyncMap[K, V]) Range(fn func(K, *V) bool) {
	sm.m.Range(func(k, v any) bool {
return fn(k.(K), v.(*V))
})
}

func (sm *SyncMap[K, V]) LoadOrStore(key K, value *V) (*V, bool) {
	actual, loaded := sm.m.LoadOrStore(key, value)
	return actual.(*V), loaded
}
