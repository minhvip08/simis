package store

import (
	"errors"
	"runtime/metrics"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/minhvip08/simis/internal/config"
	ds "github.com/minhvip08/simis/internal/datastructure"
)

// ErrOOM is returned when a write is rejected because the memory limit is
// exceeded and the eviction policy is "noeviction".
var ErrOOM = errors.New("OOM command not allowed when used memory > 'maxmemory'")

type KVStore struct {
	syncMap        ds.SyncMap[string, ds.RedisObject]
	lruTracker     *ds.LRUTracker
	totalEvictions atomic.Uint64
}

var (
	instance *KVStore
	once     sync.Once
)

func GetInstance() *KVStore {
	once.Do(func() {
		instance = &KVStore{
			syncMap:    ds.SyncMap[string, ds.RedisObject]{},
			lruTracker: ds.NewLRUTracker(),
		}
	})
	return instance
}

func (store *KVStore) Delete(key string) {
	store.lruTracker.Remove(key)
	store.syncMap.Delete(key)
}

// Store saves a key-value pair and enforces the memory limit.
// Returns ErrOOM if the eviction policy is "noeviction" and the limit is exceeded.
func (store *KVStore) Store(key string, value *ds.RedisObject) error {
	if err := store.evictIfNeeded(); err != nil {
		return err
	}
	store.syncMap.Store(key, value)
	store.lruTracker.Access(key)
	return nil
}

func (store *KVStore) GetValue(key string) (*ds.RedisObject, bool) {
	val, ok := store.syncMap.Load(key)
	if !ok {
		return nil, false
	}

	if val.TTL != nil && val.TTL.Before(time.Now()) {
		store.Delete(key)
		return nil, false
	}

	store.lruTracker.Access(key)
	return val, true
}

func (store *KVStore) GetString(key string) (string, bool) {
	val, ok := store.GetValue(key)
	if !ok {
		return "", false
	}
	return ds.AsString(val.Get())
}

func (store *KVStore) GetList(key string) (*ds.Deque, bool) {
	val, ok := store.GetValue(key)
	if !ok {
		return nil, false
	}
	return ds.AsList(val.Get())
}

func (store *KVStore) StoreList(key string, value *ds.Deque) {
	kvObj := ds.NewListObject(value)
	store.syncMap.Store(key, &kvObj)
}

// LoadOrStoreList returns the existing list for key, or creates a new one.
// Returns ErrOOM when the memory limit is exceeded with policy "noeviction".
func (store *KVStore) LoadOrStoreList(key string) (*ds.Deque, bool, error) {
	if err := store.evictIfNeeded(); err != nil {
		return nil, false, err
	}
	kvObj := ds.NewListObject(ds.NewDeque())
	val, loaded := store.syncMap.LoadOrStore(key, &kvObj)
	store.lruTracker.Access(key)
	list, ok := ds.AsList(val.Get())
	if !ok {
		return nil, false, nil
	}
	return list, loaded, nil
}

// IncrementString atomically increments a string value by 1.
// Returns the new value after increment, or an error if the value is not an integer.
// If the key doesn't exist, it's treated as "0" and incremented to "1".
func (store *KVStore) IncrementString(key string) (int, error) {
	if err := store.evictIfNeeded(); err != nil {
		return 0, err
	}

	var newCount int
	var err error

	store.syncMap.Update(key, func(old *ds.RedisObject) *ds.RedisObject {
		// Handle nil or expired keys
		if old == nil {
			newCount = 1
			kvObj := ds.NewStringObject("1")
			return &kvObj
		}

		// Check if expired
		if old.TTL != nil && old.TTL.Before(time.Now()) {
			newCount = 1
			kvObj := ds.NewStringObject("1")
			return &kvObj
		}

		// Get string value
		countStr, ok := ds.AsString(old.Get())
		if !ok {
			err = errors.New("value is not an integer or out of range")
			return old // Return old value unchanged on error
		}

		// Parse and increment
		count, parseErr := strconv.Atoi(countStr)
		if parseErr != nil {
			err = errors.New("value is not an integer or out of range")
			return old // Return old value unchanged on error
		}

		newCount = count + 1
		countStr = strconv.Itoa(newCount)
		kvObj := ds.NewStringObject(countStr)
		// Preserve TTL if it exists
		if old.TTL != nil {
			kvObj.TTL = old.TTL
		}
		return &kvObj
	})

	if err == nil {
		store.lruTracker.Access(key)
	}
	return newCount, err
}

// StoreRDBEntry stores a key-value pair from an RDB file with an optional expiry.
// This implements the RDBStore interface for loading RDB data.
func (store *KVStore) StoreRDBEntry(key, value string, expireAt *time.Time) {
	kvObj := ds.NewStringObject(value)
	if expireAt != nil {
		kvObj.TTL = expireAt
	}
	// Bypass eviction during RDB load — the store is being populated from disk.
	store.syncMap.Store(key, &kvObj)
	store.lruTracker.Access(key)
}

// GetAllKeys returns all keys in the store (non-expired only).
func (store *KVStore) GetAllKeys() []string {
	keys := make([]string, 0)
	store.syncMap.Range(func(key string, val *ds.RedisObject) bool {
		// Skip expired keys
		if val.TTL != nil && val.TTL.Before(time.Now()) {
			return true // continue iteration
		}
		keys = append(keys, key)
		return true // continue iteration
	})
	return keys
}

// LoadOrStoreStream returns the existing stream for key, or creates a new one.
// Returns ErrOOM when the memory limit is exceeded with policy "noeviction".
func (store *KVStore) LoadOrStoreStream(key string) (*ds.Stream, bool, error) {
	if err := store.evictIfNeeded(); err != nil {
		return nil, false, err
	}
	kvObj := ds.RedisObject{Value: ds.NewStream(), TTL: nil}
	val, loaded := store.syncMap.LoadOrStore(key, &kvObj)
	store.lruTracker.Access(key)
	stream, ok := ds.AsStream(val.Get())
	if !ok {
		return nil, false, nil
	}
	return stream, loaded, nil
}

// Range iterates over all key-value pairs in the store (including expired ones)
// The function receives the key and the RedisObject pointer
func (store *KVStore) Range(fn func(key string, val interface{}) bool) {
	store.syncMap.Range(func(key string, val *ds.RedisObject) bool {
		return fn(key, val)
	})
}

// LoadOrStoreSortedSet returns the existing sorted set for key, or creates a new one.
// Returns ErrOOM when the memory limit is exceeded with policy "noeviction".
func (store *KVStore) LoadOrStoreSortedSet(key string) (*ds.SortedSet, bool, error) {
	if err := store.evictIfNeeded(); err != nil {
		return nil, false, err
	}
	kvObj := ds.RedisObject{Value: ds.NewSortedSet(), TTL: nil}
	val, loaded := store.syncMap.LoadOrStore(key, &kvObj)
	store.lruTracker.Access(key)
	sortedSet, ok := ds.AsSortedSet(val.Get())
	if !ok {
		return nil, false, nil
	}
	return sortedSet, loaded, nil
}

func (store *KVStore) GetSortedSet(key string) (*ds.SortedSet, bool) {
	val, ok := store.GetValue(key)
	if !ok {
		return nil, false
	}
	return ds.AsSortedSet(val.Get())
}

func (store *KVStore) GetStream(key string) (*ds.Stream, bool) {
	val, ok := store.GetValue(key)
	if !ok {
		return nil, false
	}
	return ds.AsStream(val.Get())
}

// TotalEvictions returns the cumulative number of keys evicted due to memory pressure.
func (store *KVStore) TotalEvictions() uint64 {
	return store.totalEvictions.Load()
}

// evictIfNeeded evicts keys according to the configured policy until memory is
// within the configured limit. Returns ErrOOM if policy is "noeviction" and the
// limit is still exceeded.
func (store *KVStore) evictIfNeeded() error {
	cfg := config.GetInstance()
	if cfg.MaxMemory <= 0 {
		return nil
	}

	for currentMemory() > cfg.MaxMemory {
		if store.lruTracker.Len() == 0 {
			break
		}
		if err := store.evictOne(cfg.EvictionPolicy); err != nil {
			return err
		}
	}
	return nil
}

func (store *KVStore) evictOne(policy config.EvictionPolicy) error {
	switch policy {
	case config.PolicyNoEviction:
		return ErrOOM

	case config.PolicyAllKeysLRU:
		key, ok := store.lruTracker.PeekLRU()
		if !ok {
			return nil
		}
		store.Delete(key)
		store.totalEvictions.Add(1)

	case config.PolicyVolatileLRU:
		evictKey := ""
		store.lruTracker.IterateFromLRU(func(key string) bool {
			val, ok := store.syncMap.Load(key)
			if ok && val.TTL != nil {
				evictKey = key
				return false // stop
			}
			return true
		})
		if evictKey != "" {
			store.Delete(evictKey)
			store.totalEvictions.Add(1)
		}

	case config.PolicyAllKeysRandom:
		key, ok := store.lruTracker.PeekRandom()
		if !ok {
			return nil
		}
		store.Delete(key)
		store.totalEvictions.Add(1)

	case config.PolicyVolatileRandom:
		var evictKey string
		store.syncMap.Range(func(key string, val *ds.RedisObject) bool {
			if val.TTL != nil {
				evictKey = key
				return false // stop
			}
			return true
		})
		if evictKey != "" {
			store.Delete(evictKey)
			store.totalEvictions.Add(1)
		}

	case config.PolicyVolatileTTL:
		var minKey string
		var minTTL *time.Time
		store.syncMap.Range(func(key string, val *ds.RedisObject) bool {
			if val.TTL != nil && (minTTL == nil || val.TTL.Before(*minTTL)) {
				minKey = key
				minTTL = val.TTL
			}
			return true
		})
		if minKey != "" {
			store.Delete(minKey)
			store.totalEvictions.Add(1)
		}
	}

	return nil
}

// currentMemory returns the current heap memory in use (bytes).
// Uses runtime/metrics for a non-stop-the-world measurement.
func currentMemory() int64 {
	samples := []metrics.Sample{{Name: "/memory/classes/heap/in-use:bytes"}}
	metrics.Read(samples)
	if samples[0].Value.Kind() == metrics.KindUint64 {
		return int64(samples[0].Value.Uint64())
	}
	return 0
}

// CurrentMemory returns the current heap memory in use (bytes).
// Exported for use by the INFO command handler.
func CurrentMemory() int64 {
	return currentMemory()
}
