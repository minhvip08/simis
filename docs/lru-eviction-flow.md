# LRU Eviction — Implementation Flow

## Overview

SimiS currently implements **lazy expiration** only: expired keys are deleted on access via TTL checks in `KVStore.GetValue()`. There is no memory limit, no active eviction, and no LRU policy.

This document describes the full implementation flow for adding **LRU (Least Recently Used) eviction** with configurable memory limits and eviction policies, matching Redis's `maxmemory` / `maxmemory-policy` behavior.

---

## Current State

| Concern | Current Behavior |
|---|---|
| Key expiry | Lazy — checked on `GetValue()`, deleted if past TTL |
| Memory limit | None — unbounded growth |
| Eviction policy | None |
| Access time tracking | None |
| Active cleanup | None |

**Relevant files:**
- `internal/store/kvstore.go` — KVStore singleton, all reads/writes go here
- `internal/datastructure/valuetype.go` — `RedisObject` with `TTL *time.Time`
- `internal/config/config.go` — Server configuration struct
- `internal/handler/config.go` — CONFIG GET/SET command handler
- `internal/handler/info.go` — INFO command handler

---

## Design Decisions

### LRU Tracker Placement

The LRU tracker lives inside `KVStore` (not outside it) because:
- `KVStore` already owns all read/write access
- The tracker must be updated atomically with key operations
- The command queue guarantees sequential execution, so lock-free updates are safe on the tracker itself

### Thread Safety

Command execution is **single-threaded via the command queue** (`executor.go`). Therefore the LRU doubly-linked list needs no mutex. However, `sync.Map` within `SyncMap` still handles concurrent reads from multiple goroutines (e.g., INFO command, replication).

### Memory Measurement

Use `runtime.MemStats.HeapInuse` rather than tracking per-object sizes. This avoids error-prone manual size accounting and reflects actual Go heap pressure. Checked on every write.

### Eviction Policies Supported

| Policy | Behavior |
|---|---|
| `noeviction` | Return OOM error on writes when over limit |
| `allkeys-lru` | Evict globally least-recently-used key |
| `volatile-lru` | Evict LRU key from keys that have a TTL set |
| `allkeys-random` | Evict a random key |
| `volatile-random` | Evict a random key that has a TTL set |
| `volatile-ttl` | Evict key with the nearest (soonest) TTL expiry |

Default policy: `noeviction` (Redis default).

---

## Implementation Flow

### Phase 1 — LRU Data Structure

**New file:** `internal/datastructure/lru.go`

Implement a **doubly linked list + hash map** LRU tracker. The list stores keys in access order (head = most recent, tail = least recent). The map provides O(1) node lookup by key.

```
LRUTracker
├── head *lruNode   (most recently used)
├── tail *lruNode   (least recently used)
├── nodes map[string]*lruNode
└── size  int

lruNode
├── key  string
├── prev *lruNode
└── next *lruNode
```

**Methods:**

| Method | Description |
|---|---|
| `NewLRUTracker() *LRUTracker` | Constructor |
| `Access(key string)` | Move key to head (called on every read AND write) |
| `Add(key string)` | Insert new key at head |
| `Remove(key string)` | Delete key from tracker |
| `EvictLRU() (string, bool)` | Remove and return tail key (LRU candidate) |
| `EvictRandom() (string, bool)` | Return any key from the map |
| `Len() int` | Current tracked key count |

**Implementation notes:**
- `Access()` calls `Remove()` + `Add()` internally
- `Add()` is a no-op if the key already exists (caller should call `Access()` instead)
- All methods are **not** goroutine-safe — callers own synchronization (safe here because executor is single-threaded for writes)

---

### Phase 2 — Config Extension

**File:** `internal/config/config.go`

Add two new fields to the `Config` struct:

```go
type Config struct {
    // ... existing fields ...
    MaxMemory      int64          // Maximum memory in bytes; 0 = unlimited
    EvictionPolicy EvictionPolicy // What to do when MaxMemory is reached
}

type EvictionPolicy string

const (
    PolicyNoEviction     EvictionPolicy = "noeviction"
    PolicyAllKeysLRU     EvictionPolicy = "allkeys-lru"
    PolicyVolatileLRU    EvictionPolicy = "volatile-lru"
    PolicyAllKeysRandom  EvictionPolicy = "allkeys-random"
    PolicyVolatileRandom EvictionPolicy = "volatile-random"
    PolicyVolatileTTL    EvictionPolicy = "volatile-ttl"
)
```

**CLI flags** (add to `cmd/server/main.go`):

```
--maxmemory <bytes>               Memory limit; 0 = disabled (default: 0)
--maxmemory-policy <policy>       Eviction policy (default: noeviction)
```

**Parse helper:** Support human-readable sizes in `--maxmemory` (e.g., `100mb`, `1gb`) by adding a `parseMemorySize(s string) (int64, error)` utility in `internal/utils/`.

---

### Phase 3 — KVStore Integration

**File:** `internal/store/kvstore.go`

#### 3a. Add LRU Tracker Field

```go
type KVStore struct {
    syncMap    ds.SyncMap[string, ds.RedisObject]
    lruTracker *ds.LRUTracker
}
```

Initialize `lruTracker` in the `once.Do()` constructor.

#### 3b. Update `Store(key, value)`

After storing to `syncMap`:
1. Call `lruTracker.Access(key)` (if key existed) or `lruTracker.Add(key)` (new key)
2. Call `evictIfNeeded()` to enforce memory limit

```
Store(key, value)
    ├── syncMap.Store(key, value)
    ├── lruTracker.Access(key)    // update recency
    └── evictIfNeeded()           // enforce limit
```

#### 3c. Update `GetValue(key)`

After loading from `syncMap` (before returning):
1. If found and not expired: call `lruTracker.Access(key)` to refresh position
2. If expired: call `lruTracker.Remove(key)` and delete as before

```
GetValue(key)
    ├── syncMap.Load(key)
    ├── if expired → lruTracker.Remove(key) + syncMap.Delete(key) → return nil
    └── if found  → lruTracker.Access(key) → return value
```

#### 3d. Update `Delete(key)`

Call `lruTracker.Remove(key)` alongside `syncMap.Delete(key)`.

#### 3e. Add `evictIfNeeded()`

```
evictIfNeeded()
    ├── cfg = config.GetInstance()
    ├── if cfg.MaxMemory == 0 → return (no limit)
    ├── loop while currentMemory() > cfg.MaxMemory:
    │   └── evictOne(cfg.EvictionPolicy)
    └── (exits when under limit or no evictable key found)
```

`evictOne(policy EvictionPolicy)`:

| Policy | Action |
|---|---|
| `noeviction` | Return OOM error (propagated to caller, write is rejected) |
| `allkeys-lru` | `key = lruTracker.EvictLRU()` → `Delete(key)` |
| `volatile-lru` | Walk from LRU tail; find first key with TTL set → `Delete(key)` |
| `allkeys-random` | `key = lruTracker.EvictRandom()` → `Delete(key)` |
| `volatile-random` | Random walk; find key with TTL → `Delete(key)` |
| `volatile-ttl` | Scan keys with TTL; pick nearest expiry → `Delete(key)` |

#### 3f. Add `currentMemory() int64`

```go
func currentMemory() int64 {
    var ms runtime.MemStats
    runtime.ReadMemStats(&ms)
    return int64(ms.HeapInuse)
}
```

`runtime.ReadMemStats` triggers a STW pause; only call it inside `evictIfNeeded()` (not on every read). This is acceptable overhead at write time.

#### 3g. OOM Error Propagation for `noeviction`

`evictIfNeeded()` returns an error when policy is `noeviction` and over limit. `Store()` returns this error. Handler callers must check and return a Redis OOM error response:

```
-OOM command not allowed when used memory > 'maxmemory'.\r\n
```

This requires changing the `Store()` signature from `void` to `error`. Update all call sites in handlers accordingly.

---

### Phase 4 — CONFIG Command Update

**File:** `internal/handler/config.go`

The existing `CONFIG GET` and `CONFIG SET` handlers must recognize the new fields:

**CONFIG GET maxmemory** → returns current `MaxMemory` as string (bytes)
**CONFIG GET maxmemory-policy** → returns current `EvictionPolicy`
**CONFIG SET maxmemory <bytes>** → updates `cfg.MaxMemory` at runtime
**CONFIG SET maxmemory-policy <policy>** → validates and updates `cfg.EvictionPolicy`

Validation for `CONFIG SET maxmemory-policy`: reject unknown policy names with a descriptive error.

---

### Phase 5 — INFO Command Update

**File:** `internal/handler/info.go`

Add a `# Memory` section to INFO output:

```
# Memory
used_memory:<HeapInuse bytes>
maxmemory:<cfg.MaxMemory>
maxmemory_policy:<cfg.EvictionPolicy>
mem_evictions:<total eviction count>
```

Track `totalEvictions uint64` as an atomic counter in `KVStore`, incremented in `evictOne()`.

---

### Phase 6 — Background Eviction Goroutine (Optional)

For `volatile-ttl` and memory pressure scenarios, add an optional background eviction goroutine:

**File:** `internal/store/eviction.go` (new file)

```
StartEvictionWorker(interval time.Duration)
    loop every interval:
        if currentMemory() > cfg.MaxMemory * 0.95:
            evictBatch(N=10)   // evict up to 10 keys per tick
```

Start this goroutine in `GetInstance()` (the KVStore constructor) if `MaxMemory > 0`.

**Caution:** This goroutine runs concurrently with the command queue. For policies that walk the LRU list (`volatile-lru`, `volatile-ttl`), add a `sync.Mutex` to `LRUTracker` (only needed when background eviction is enabled).

---

## Data Flow After Changes

```
Client WRITE command (SET, LPUSH, ZADD, etc.)
        │
        ▼
SessionHandler → CommandQueue
        │
        ▼
Executor.executeCommand()
        │
        ▼
Handler.Execute()
        │
        ├── Parse arguments
        ├── Construct RedisObject
        └── KVStore.Store(key, obj)
                │
                ├── syncMap.Store(key, obj)         // persist value
                ├── lruTracker.Access/Add(key)       // update LRU order
                └── evictIfNeeded()                  // enforce memory limit
                        │
                        ├── currentMemory() > maxMemory?
                        │       NO → return nil
                        │       YES → evictOne(policy)
                        │               │
                        │               ├── allkeys-lru:     lruTracker.EvictLRU() → Delete()
                        │               ├── volatile-lru:    walk tail → Delete() if TTL set
                        │               ├── allkeys-random:  lruTracker.EvictRandom() → Delete()
                        │               ├── volatile-random: random walk → Delete() if TTL set
                        │               ├── volatile-ttl:    find min TTL → Delete()
                        │               └── noeviction:      return OOM error
                        │
                        └── repeat until under limit or no evictable key
        │
        ▼
Handler returns OOM error or success response
```

```
Client READ command (GET, LRANGE, ZRANGE, etc.)
        │
        ▼
KVStore.GetValue(key)
        │
        ├── syncMap.Load(key)
        ├── if expired → lruTracker.Remove(key) + syncMap.Delete() → nil
        └── if found   → lruTracker.Access(key) → return *RedisObject
```

---

## File Change Summary

| File | Change Type | Description |
|---|---|---|
| `internal/datastructure/lru.go` | New | LRUTracker (doubly linked list + map) |
| `internal/config/config.go` | Modify | Add `MaxMemory`, `EvictionPolicy` fields |
| `internal/utils/memory.go` | New | `parseMemorySize()` helper |
| `cmd/server/main.go` | Modify | Add `--maxmemory`, `--maxmemory-policy` CLI flags |
| `internal/store/kvstore.go` | Modify | Add `lruTracker`, update Store/GetValue/Delete, add `evictIfNeeded()` |
| `internal/store/eviction.go` | New (optional) | Background eviction worker goroutine |
| `internal/handler/config.go` | Modify | CONFIG GET/SET for maxmemory, maxmemory-policy |
| `internal/handler/info.go` | Modify | Add `# Memory` section to INFO output |
| `internal/constants/constants.go` | Modify | Register write commands that must check OOM |

---

## Implementation Order

1. **`internal/datastructure/lru.go`** — standalone, no deps, easy to unit test
2. **`internal/config/config.go`** + **`cmd/server/main.go`** — config before KVStore uses it
3. **`internal/utils/memory.go`** — small helper, no deps
4. **`internal/store/kvstore.go`** — core integration; depends on LRU + Config
5. **`internal/handler/config.go`** — runtime config update support
6. **`internal/handler/info.go`** — observability
7. **`internal/store/eviction.go`** — background worker (optional, add last)

---

## Testing Plan

| Test | Location | Validates |
|---|---|---|
| LRU order correctness | `internal/datastructure/lru_test.go` | Access → tail becomes head; EvictLRU returns oldest |
| LRU eviction on write | `internal/store/kvstore_test.go` | Writing beyond maxmemory triggers eviction |
| noeviction policy | `internal/store/kvstore_test.go` | Write returns OOM error when over limit |
| volatile-lru skips no-TTL keys | `internal/store/kvstore_test.go` | Keys without TTL are not evicted |
| CONFIG SET maxmemory | Integration test | Runtime update takes effect on next write |
| INFO memory section | Integration test | Memory stats present and non-zero |

---

## Redis Compatibility Notes

- `CONFIG GET maxmemory` must return bytes as a bulk string (not integer)
- `CONFIG SET maxmemory 0` disables the limit (same as Redis)
- OOM error string must match exactly: `-OOM command not allowed when used memory > 'maxmemory'.`
- `INFO memory` field `used_memory` in Redis is `malloc`-level; using `HeapInuse` is an approximation. Label it as `used_memory_heap` to avoid confusion with Redis clients that parse exact field names.
- Write commands that should be blocked under `noeviction`: all commands in `constants.WriteCommands` plus `EXPIRE`, `EXPIREAT`, `PEXPIRE`, `PEXPIREAT` (if those get implemented).
