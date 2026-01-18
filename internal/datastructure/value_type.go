package datastructure

import (
	"time"
)

type RedisObject struct {
	Value RValue
	TTL   *time.Time
}

func (o *RedisObject) SetExpiry(expiry time.Duration) {
	t := time.Now().Add(expiry)
	o.TTL = &t
}

// NewStringObject creates a new RedisObject containing a StringValue.
func NewStringObject(value string) RedisObject {
	return RedisObject{Value: NewStringValue(value), TTL: nil}
}

func NewListObject(value *Deque) RedisObject {
	return RedisObject{Value: value, TTL: nil}
}

func (o *RedisObject) Get() RValue {
	return o.Value
}
