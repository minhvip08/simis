package pubsub

import (
	"sync"

	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/utils"
)

type PubSubManager struct {
	mu            sync.RWMutex
	channels      map[string]map[*connection.RedisConnection]bool // channel name -> set of subscribed connections
	subscriptions map[string]map[string]bool                      // connection ID -> set of subscribed channel names
}

var (
	manager *PubSubManager
	once    sync.Once
)

func GetPubSubManager() *PubSubManager {
	once.Do(func() {
		manager = &PubSubManager{
			channels:      make(map[string]map[*connection.RedisConnection]bool),
			subscriptions: make(map[string]map[string]bool),
		}
	})
	return manager
}

func (m *PubSubManager) Subscribe(conn *connection.RedisConnection, channel string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn.EnterPubSubMode()
	if _, exists := m.channels[channel]; !exists {
		m.channels[channel] = make(map[*connection.RedisConnection]bool)
	}
	m.channels[channel][conn] = true

	if _, exists := m.subscriptions[conn.ID]; !exists {
		m.subscriptions[conn.ID] = make(map[string]bool)
	}
	m.subscriptions[conn.ID][channel] = true

	return len(m.subscriptions[conn.ID])
}

func (m *PubSubManager) Unsubscribe(conn *connection.RedisConnection, channel string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conns, exists := m.channels[channel]; exists {
		delete(conns, conn)
	}
	if subs, exists := m.subscriptions[conn.ID]; exists {
		delete(subs, channel)
		if len(subs) == 0 {
			conn.ExitPubSubMode()
		}
	}

	return len(m.subscriptions[conn.ID])
}

func (m *PubSubManager) Publish(channel string, message string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if conns, exists := m.channels[channel]; exists {
		for conn := range conns {
			conn.SendResponse(
				utils.ToSimpleRespArray([]string{
					utils.ToBulkString("message"),
					utils.ToBulkString(channel),
					utils.ToBulkString(message),
				}),
			)
		}
	}
	return len(m.channels[channel])
}
