package datastructure

import "fmt"

type Deque struct {
	data []string
}

func (d *Deque) Kind() RType {
	return RList
}

func NewDeque() *Deque {
	return &Deque{data: make([]string, 0)}
}

func AsList(v RValue) (*Deque, bool) {
	if lv, ok := v.(*Deque); ok {
		return lv, true
	}
	return nil, false
}

func (d *Deque) PushFront(items ...string) int {
	d.data = append(items, d.data...)
	return len(d.data)
}

func (d *Deque) PushBack(items ...string) int {
	d.data = append(d.data, items...)
	return len(d.data)
}

// Try to pop an element from the front of the deque.
func (d *Deque) PopFront() (string, bool) {
	if len(d.data) == 0 {
		return "", false
	}
	val := d.data[0]
	d.data = d.data[1:]
	return val, true
}

func (d *Deque) PopBack() (string, bool) {
	if len(d.data) == 0 {
		return "", false
	}
	val := d.data[len(d.data)-1]
	d.data = d.data[:len(d.data)-1]
	return val, true
}

func (d *Deque) PopNFront(n int) ([]string, int) {
	if n > len(d.data) {
		n = len(d.data)
	}
	values := d.data[:n]
	d.data = d.data[n:]
	return values, n
}

func (d *Deque) Length() int {
	return len(d.data)
}

// LRange returns the elements between the left and right indices (inclusive).
func (d *Deque) LRange(left, right int) []string {
	n := len(d.data)
	if n == 0 {
		return []string{}
	}

	if left < 0 {
		left = n + left
	}
	if right < 0 {
		right = n + right
	}
	
	// Clamp to valid range
	right = max(0, min(right, n-1))
	left = max(0, min(left, n-1))

	if left > right {
		return []string{}
	}

	return d.data[left : right+1]
}


func (d *Deque) Clear() {
	d.data = []string{}
}

func (d *Deque) String() string {
	return fmt.Sprintf("Deque{%v}", d.data)
}

