package datastructure

import (
	"cmp"
	"sync"
)

type KeyValue struct {
	Key   string
	Value float64
}

func (kv KeyValue) Compare(other KeyValue) int {
	if kv.Value == other.Value {
		return cmp.Compare(kv.Key, other.Key)
	}
	return cmp.Compare(kv.Value, other.Value)
}

type SortedSet struct {
	mu       sync.RWMutex
	skipList *SkipList[KeyValue]
	score    map[string]float64
}

func NewSortedSet() *SortedSet {
	return &SortedSet{
		skipList: NewComparableSkipList[KeyValue](),
		score:    make(map[string]float64),
	}
}

func (ss *SortedSet) Kind() RType {
	return RSortedSet
}

func AsSortedSet(val RValue) (*SortedSet, bool) {
	if sv, ok := val.(*SortedSet); ok {
		return sv, true
	}
	return nil, false
}

func (ss *SortedSet) Lock() {
	ss.mu.Lock()
}

func (ss *SortedSet) Unlock() {
	ss.mu.Unlock()
}

func (ss *SortedSet) RLock() {
	ss.mu.RLock()
}

func (ss *SortedSet) RUnlock() {
	ss.mu.RUnlock()
}

func (ss *SortedSet) Add(arg []KeyValue) int {
	ss.Lock()
	defer ss.Unlock()

	count := 0
	for _, kv := range arg {
		if _, ok := ss.score[kv.Key]; !ok {
			count++
		} else {
			ss.skipList.Delete(KeyValue{
				Key:   kv.Key,
				Value: ss.score[kv.Key],
			})
		}
		ss.skipList.Add(kv)
		ss.score[kv.Key] = kv.Value
	}

	return count
}

func (ss *SortedSet) GetRank(key string) int {
	ss.RLock()
	defer ss.RUnlock()

	score, ok := ss.score[key]
	if !ok {
		return -1
	}

	rank, _ := ss.skipList.GetRank(KeyValue{
		Key:   key,
		Value: score,
	})
	return rank - 1
}

func (ss *SortedSet) GetRange(start, end int) []KeyValue {
	ss.RLock()
	defer ss.RUnlock()

	if start >= ss.skipList.Len() {
		return []KeyValue{}
	}

	if end >= ss.skipList.Len() {
		end = ss.skipList.Len() - 1
	}

	if start < 0 {
		start = max(0, start+ss.skipList.Len())
	}

	if end < 0 {
		end = max(0, end+ss.skipList.Len())
	}

	if start > end {
		return []KeyValue{}
	}

	node, ok := ss.skipList.SearchByRank(start + 1)
	if !ok {
		return []KeyValue{}
	}

	len := end - start + 1
	result := make([]KeyValue, len)
	for i := 0; i < len && node != nil; i++ {
		result[i] = node.Value()
		node = node.GetNextNodeAtLevel(0)
	}

	return result
}


func (ss *SortedSet) GetRangeByValue(start float64, end float64) []KeyValue {
	ss.RLock()
	defer ss.RUnlock()

	if start > end {
		return []KeyValue{}
	}

	node, ok := ss.skipList.GetLowerBound(KeyValue{Key: "", Value: start})
	if !ok {
		return []KeyValue{}
	}

	result := make([]KeyValue, 0)
	for ; node != nil && node.Value().Value <= end; node = node.GetNextNodeAtLevel(0) {
		result = append(result, node.Value())
	}

	return result
}

func (ss *SortedSet) GetCardinality() int {
	ss.RLock()
	defer ss.RUnlock()
	
	return ss.skipList.Len()
}

func (ss *SortedSet) GetScore(key string) (float64, bool) {
	ss.RLock()
	defer ss.RUnlock()

	score, ok := ss.score[key]
	return score, ok
}

func (ss *SortedSet) Remove(keys ...string) int {
	ss.Lock()
	defer ss.Unlock()

	count := 0
	for _, key := range keys {
		if _, ok := ss.score[key]; !ok {
			continue
		}

		ss.skipList.Delete(KeyValue{Key: key, Value: ss.score[key]})
		delete(ss.score, key)
		count++
	}
	return count
}
