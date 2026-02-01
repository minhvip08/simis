package replication

import "github.com/minhvip08/simis/internal/connection"

type ReplicaInfo struct {
	Connection      *connection.RedisConnection
	ListeningPort   int
	Capabilities    []string
	IsHandshakeDone bool
	LastAckOffset   int
}

func NewReplicaInfo(conn *connection.RedisConnection) *ReplicaInfo {
	return &ReplicaInfo{
		Connection:      conn,
		Capabilities:    make([]string, 0),
		IsHandshakeDone: false,
		LastAckOffset:   0,
	}
}

// SetHandshakeDone sets the handshake done flag
func (r *ReplicaInfo) SetHandshakeDone() {
	r.IsHandshakeDone = true
}

// AddCapability adds a capability to the replica
func (r *ReplicaInfo) AddCapability(capability string) {
	r.Capabilities = append(r.Capabilities, capability)
}

// HasCapability checks if the replica has a specific capability
func (r *ReplicaInfo) HasCapability(capability string) bool {
	for _, cap := range r.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}
