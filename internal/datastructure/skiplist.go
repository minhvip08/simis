// Package skiplist provides a generic probabilistic skip list data structure.
//
// A skip list is a data structure that allows fast search, insertion, and deletion
// operations with O(log n) average time complexity. It achieves this by maintaining
// multiple layers of linked lists with "express lanes" at higher levels.
//
// This implementation supports:
//   - Generic types with ordered constraints or custom comparators
//   - Rank-based queries (search by position)
//   - No duplicates (set semantics)
//   - Deterministic insertion at specific levels (for testing)
//
// Example usage:
//
//	sl := skiplist.NewSkipList[int]()
//	sl.Add(10)
//	sl.Add(5)
//	sl.Add(20)
//
//	if node, found := sl.SearchByValue(10); found {
//	    fmt.Println("Found:", node)
//	}
//
//	if node, found := sl.SearchByRank(2); found {
//	    fmt.Println("Second element:", node)
//	}
package datastructure

import (
	"cmp"
	"math/rand"
)

const (
	MaxLevelCap         = 16
	Probability float32 = 0.5
)

type Comparator[T any] func(a, b T) int

type Comparable[T any] interface {
	Compare(other T) int
}

type Node[T any] struct {
	val     T
	skips   []int
	forward []*Node[T]
}

// Value returns the value stored in the node.
func (n *Node[T]) Value() T {
	return n.val
}

func (n *Node[T]) GetNextNodeAtLevel(level int) *Node[T] {
	if level >= len(n.forward) {
		return nil
	}
	return n.forward[level]
}

type SkipList[T any] struct {
	head       *Node[T]
	tail       *Node[T]
	maxLevel   int
	length     int
	levelCount [MaxLevelCap + 1]int
	comparator Comparator[T]
}

func NewNode[T any](val T, forwards int) *Node[T] {
	return &Node[T]{
		val:     val,
		forward: make([]*Node[T], forwards),
		skips:   make([]int, forwards),
	}
}

func NewSkipList[T cmp.Ordered]() *SkipList[T] {
	var zero T
	first := NewNode(zero, MaxLevelCap+1)

	for i := 0; i <= MaxLevelCap; i++ {
		first.forward[i] = nil
		first.skips[i] = 1
	}

	return &SkipList[T]{
		head:       first,
		tail:       nil,
		maxLevel:   0,
		length:     0,
		comparator: cmp.Compare[T],
	}
}

func NewComparableSkipList[T Comparable[T]]() *SkipList[T] {
	var zero T
	first := NewNode(zero, MaxLevelCap+1)

	for i := 0; i <= MaxLevelCap; i++ {
		first.forward[i] = nil
		first.skips[i] = 1
	}

	return &SkipList[T]{
		head:     first,
		tail:     nil,
		maxLevel: 0,
		length:   0,
		comparator: func(a, b T) int {
			return a.Compare(b)
		},
	}
}

func randomLevel() int {
	lvl := 0
	for rand.Float32() >= Probability && lvl < MaxLevelCap {
		lvl++
	}
	return lvl
}

func (sl *SkipList[T]) Add(val T) {
	sl.InsertAtLevel(val, randomLevel())
}

func (sl *SkipList[T]) InsertAtLevel(val T, lvl int) {
	hierarchy := [MaxLevelCap + 1]*Node[T]{}
	rank := [MaxLevelCap + 1]int{}
	curr := sl.head
	skipped := 0

	for currLevel := sl.maxLevel; currLevel >= 0; currLevel-- {
		for curr.forward[currLevel] != nil && sl.comparator(curr.forward[currLevel].val, val) <= 0 {
			skipped += curr.skips[currLevel]
			curr = curr.forward[currLevel]
		}

		// do nothing if the value is already added
		if curr != sl.head && sl.comparator(curr.val, val) == 0 {
			return
		}

		hierarchy[currLevel] = curr
		rank[currLevel] = skipped
	}

	if lvl > sl.maxLevel {
		for i := sl.maxLevel + 1; i <= lvl; i++ {
			hierarchy[i] = sl.head
			rank[i] = 0
		}

		sl.maxLevel = lvl
	}

	newNode := NewNode(val, lvl+1)

	for i := 0; i <= lvl; i++ {
		newNode.forward[i] = hierarchy[i].forward[i]
		hierarchy[i].forward[i] = newNode

		newNode.skips[i] = rank[i] + hierarchy[i].skips[i] - skipped
		hierarchy[i].skips[i] = skipped - rank[i] + 1

		sl.levelCount[i]++
	}

	for i := lvl + 1; i <= sl.maxLevel; i++ {
		hierarchy[i].skips[i]++
	}
	for i := sl.maxLevel + 1; i <= MaxLevelCap; i++ {
		sl.head.skips[i]++
	}

	sl.length++
}

func (sl *SkipList[T]) Delete(val T) {
	hierarchy := [MaxLevelCap + 1]*Node[T]{}
	curr := sl.head
	var nodeToDelete *Node[T] = nil

	for currLevel := sl.maxLevel; currLevel >= 0; currLevel-- {
		for curr.forward[currLevel] != nil && sl.comparator(curr.forward[currLevel].val, val) < 0 {
			curr = curr.forward[currLevel]
		}

		nodeToDelete = curr.forward[currLevel]
		if nodeToDelete != nil && sl.comparator(nodeToDelete.val, val) == 0 {
			curr.skips[currLevel] += nodeToDelete.skips[currLevel] - 1
			curr.forward[currLevel] = nodeToDelete.forward[currLevel]
			nodeToDelete.forward[currLevel] = nil

			sl.levelCount[currLevel]--
			if sl.levelCount[currLevel] == 0 {
				sl.maxLevel--
			}
		} else {
			hierarchy[currLevel] = curr
		}
	}

	// if the node to delete was found only then reduce the span of the remaining hierarchy
	if nodeToDelete == nil {
		return
	}

	currLevel := len(nodeToDelete.skips)
	for ; currLevel <= sl.maxLevel; currLevel++ {
		hierarchy[currLevel].skips[currLevel]--
	}
	for ; currLevel <= MaxLevelCap; currLevel++ {
		sl.head.skips[currLevel]--
	}

	sl.length--
}

func (sl *SkipList[T]) SearchByValue(val T) (*Node[T], bool) {
	curr := sl.head

	for currLevel := sl.maxLevel; currLevel >= 0; currLevel-- {
		for curr.forward[currLevel] != nil && sl.comparator(curr.forward[currLevel].val, val) <= 0 {
			curr = curr.forward[currLevel]
		}

		if curr != sl.head && sl.comparator(curr.val, val) == 0 {
			return curr, true
		}
	}
	return nil, false
}

func (sl *SkipList[T]) SearchByRank(rank int) (*Node[T], bool) {
	if rank < 1 || rank > sl.length {
		return nil, false
	}

	curr := sl.head
	rankUntil := 0

	for currLevel := sl.maxLevel; currLevel >= 0; currLevel-- {
		for curr != nil && rankUntil+curr.skips[currLevel] <= rank {
			rankUntil += curr.skips[currLevel]
			curr = curr.forward[currLevel]
		}

		if rankUntil == rank {
			return curr, true
		}
	}
	return nil, false
}

func (sl *SkipList[T]) GetLowerBound(val T) (*Node[T], bool) {
	curr := sl.head

	for currLevel := sl.maxLevel; currLevel >= 0; currLevel-- {
		for curr.forward[currLevel] != nil && sl.comparator(curr.forward[currLevel].val, val) < 0 {
			curr = curr.forward[currLevel]
		}
	}

	// move to the next node
	curr = curr.forward[0]

	if curr != nil {
		return curr, true
	}

	return nil, false
}

func (sl *SkipList[T]) GetRank(item T) (int, bool) {
	curr := sl.head
	rank := 0

	for currLevel := sl.maxLevel; currLevel >= 0; currLevel-- {
		for curr.forward[currLevel] != nil && sl.comparator(curr.forward[currLevel].val, item) <= 0 {
			rank += curr.skips[currLevel]
			curr = curr.forward[currLevel]
		}

		if curr != sl.head && sl.comparator(curr.val, item) == 0 {
			return rank, true
		}
	}
	return -1, false
}

// Len returns the number of elements in the skip list.
func (sl *SkipList[T]) Len() int {
	return sl.length
}

// Contains checks if a value exists in the skip list.
func (sl *SkipList[T]) Contains(val T) bool {
	_, found := sl.SearchByValue(val)
	return found
}

// Range iterates over all elements in the skip list in ascending order.
// The function fn is called for each element. If fn returns false, iteration stops.
func (sl *SkipList[T]) Range(fn func(val T) bool) {
	for curr := sl.head.forward[0]; curr != nil && curr != sl.tail; curr = curr.forward[0] {
		if !fn(curr.val) {
			break
		}
	}
}

// Clear removes all elements from the skip list.
func (sl *SkipList[T]) Clear() {
	var zero T
	sl.head = NewNode(zero, MaxLevelCap+1)
	for i := 0; i <= MaxLevelCap; i++ {
		sl.head.forward[i] = nil
		sl.head.skips[i] = 1
	}
	sl.tail = nil
	sl.maxLevel = 0
	sl.length = 0
	sl.levelCount = [MaxLevelCap + 1]int{}
}

// IsEmpty returns true if the skip list contains no elements.
func (sl *SkipList[T]) IsEmpty() bool {
	return sl.length == 0
}
