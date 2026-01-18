package datastructure

import "sync"

// SyncMap: Safe for concurrent use.
// Warning: Returned *V should be treated as Read-Only.
// To modify, use Update() or Store() a new object.
type SyncMap[K comparable, V any] struct {
	m sync.Map // map[K]*entry[V]
}

type entry[V any] struct {
	mu  sync.RWMutex
	val *V
}

func (sm *SyncMap[K, V]) Store(key K, value *V) {
	// Fast path: update existing entry
	if eVal, ok := sm.m.Load(key); ok {
		e := eVal.(*entry[V])
		e.mu.Lock()
		e.val = value
		e.mu.Unlock()
		return
	}

	// Slow path: create new entry with value already set
	newE := &entry[V]{val: value}
	actual, loaded := sm.m.LoadOrStore(key, newE)

	if loaded {
		// Lost race: someone else created entry first, update it
		e := actual.(*entry[V])
		e.mu.Lock()
		e.val = value
		e.mu.Unlock()
	}
	// If we won the race, entry is already in map with correct value
}

// Load returns the pointer to value.
func (sm *SyncMap[K, V]) Load(key K) (*V, bool) {
	eVal, ok := sm.m.Load(key)
	if !ok {
		return nil, false
	}
	e := eVal.(*entry[V])

	e.mu.RLock()
	val := e.val
	e.mu.RUnlock()
	return val, true
}

// Update guarantees atomic read-modify-write.
// fn can receive nil if key is new.
func (sm *SyncMap[K, V]) Update(key K, fn func(old *V) *V) {
	// Fast path: update existing entry
	if eVal, ok := sm.m.Load(key); ok {
		e := eVal.(*entry[V])
		e.mu.Lock()
		e.val = fn(e.val)
		e.mu.Unlock()
		return
	}

	// Slow path: create new entry with computed value
	newVal := fn(nil)
	newE := &entry[V]{val: newVal}
	actual, loaded := sm.m.LoadOrStore(key, newE)

	if loaded {
		// Lost race: someone else created entry first, update it
		e := actual.(*entry[V])
		e.mu.Lock()
		e.val = fn(e.val)
		e.mu.Unlock()
	}
}

func (sm *SyncMap[K, V]) Delete(key K) {
	sm.m.Delete(key)
}

func (sm *SyncMap[K, V]) Range(fn func(K, *V) bool) {
	sm.m.Range(func(k, v any) bool {
		key := k.(K)
		e := v.(*entry[V])

		e.mu.RLock()
		val := e.val
		e.mu.RUnlock()

		return fn(key, val)
	})
}

func (sm *SyncMap[K, V]) LoadOrStore(key K, value *V) (*V, bool) {
	// 1. Try Load
	if actual, ok := sm.m.Load(key); ok {
		e := actual.(*entry[V])
		e.mu.RLock()
		val := e.val
		e.mu.RUnlock()
		return val, true
	}

	// 2. Alloc new entry
	newE := &entry[V]{val: value}
	actual, loaded := sm.m.LoadOrStore(key, newE)

	if loaded {
		e := actual.(*entry[V])
		e.mu.RLock()
		val := e.val
		e.mu.RUnlock()
		return val, true
	}
	return value, false
}
