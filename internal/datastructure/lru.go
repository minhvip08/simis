package datastructure

// lruNode is a doubly-linked list node used by LRUTracker.
type lruNode struct {
	key  string
	prev *lruNode
	next *lruNode
}

// LRUTracker tracks key access order for LRU eviction.
// Head = most recently used, Tail = least recently used.
// Not goroutine-safe; safe to use without locking because KVStore writes
// are serialized through the executor's command queue.
type LRUTracker struct {
	head  *lruNode
	tail  *lruNode
	nodes map[string]*lruNode
	size  int
}

func NewLRUTracker() *LRUTracker {
	return &LRUTracker{
		nodes: make(map[string]*lruNode),
	}
}

// Access marks a key as most recently used, moving it to the head.
// If the key is not yet tracked, it is added.
func (l *LRUTracker) Access(key string) {
	if node, ok := l.nodes[key]; ok {
		l.removeFromList(node)
		l.addToHead(node)
		return
	}
	l.insertNew(key)
}

// Add inserts a new key as most recently used.
// If the key is already tracked, it is treated as an Access.
func (l *LRUTracker) Add(key string) {
	l.Access(key)
}

// Remove deletes a key from the tracker. No-op if the key is not tracked.
func (l *LRUTracker) Remove(key string) {
	if node, ok := l.nodes[key]; ok {
		l.removeFromList(node)
		delete(l.nodes, key)
		l.size--
	}
}

// PeekLRU returns the least recently used key without removing it.
func (l *LRUTracker) PeekLRU() (string, bool) {
	if l.tail == nil {
		return "", false
	}
	return l.tail.key, true
}

// PeekRandom returns any key from the tracker without removing it.
func (l *LRUTracker) PeekRandom() (string, bool) {
	for key := range l.nodes {
		return key, true
	}
	return "", false
}

// IterateFromLRU calls fn for each key starting from the least recently used
// toward the most recently used. Return false from fn to stop early.
// The list must not be structurally modified during iteration.
func (l *LRUTracker) IterateFromLRU(fn func(key string) bool) {
	node := l.tail
	for node != nil {
		toward := node.prev // direction toward head (MRU)
		if !fn(node.key) {
			return
		}
		node = toward
	}
}

// Len returns the number of tracked keys.
func (l *LRUTracker) Len() int {
	return l.size
}

func (l *LRUTracker) insertNew(key string) {
	node := &lruNode{key: key}
	l.nodes[key] = node
	l.addToHead(node)
	l.size++
}

func (l *LRUTracker) addToHead(node *lruNode) {
	node.prev = nil
	node.next = l.head
	if l.head != nil {
		l.head.prev = node
	}
	l.head = node
	if l.tail == nil {
		l.tail = node
	}
}

func (l *LRUTracker) removeFromList(node *lruNode) {
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		l.head = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	} else {
		l.tail = node.prev
	}
	node.prev = nil
	node.next = nil
}
