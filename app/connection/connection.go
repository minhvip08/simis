package connection

import (
	"net"

	"github.com/google/uuid"
	err "github.com/minhvip08/simis/app/error"
	"github.com/minhvip08/simis/app/logger"
	"github.com/minhvip08/simis/app/utils"
)

type RedisConnection struct {
	ID               string
	conn             net.Conn
	inTransaction    bool
	queuedCommands   []Command
	suppressResponse bool
	isMaster         bool
	inPubSubMode     bool
	username         string // Username for access control
	authenticated    bool
}

type Command struct {
	Command  string
	Args     []string
	Response chan string
}

func NewRedisConnection(conn net.Conn) *RedisConnection {
	// TODO: ACL initialization
	return &RedisConnection{
		ID:               uuid.New().String(),
		conn:             conn,
		inTransaction:    false,
		queuedCommands:   make([]Command, 0),
		suppressResponse: false,
		isMaster:         false,
		username:         "default", // Default user for unauthenticated connections
		authenticated:    true,      // TODO: Set to false when ACLs are implemented
	}
}

func CreateCommand(command string, args ...string) Command {
	return Command{
		Command:  command,
		Args:     args,
		Response: make(chan string),
	}
}

func (conn *RedisConnection) EnqueueCommand(command Command) (bool, error) {
	if conn.IsInTransaction() {
		conn.queuedCommands = append(conn.queuedCommands, command)
		return true, nil
	}
	return false, err.ErrNotInTransaction
}

func (conn *RedisConnection) DequeueCommand() (Command, bool) {
	if len(conn.queuedCommands) == 0 {
		return Command{}, false
	}
	return conn.queuedCommands[0], true
}

func (conn *RedisConnection) Close() error {
	return conn.conn.Close()
}

func (conn *RedisConnection) Read(p []byte) (int, error) {
	return conn.conn.Read(p)
}

func (conn *RedisConnection) SendResponse(message string) {
	if conn.suppressResponse {
		return
	}
	_, writeErr := conn.conn.Write([]byte(message))
	if writeErr != nil {
		logger.Error("Error writing response", "error", writeErr, "address", conn.GetAddress())
	}
}

func (conn *RedisConnection) IsInTransaction() bool {
	return conn.inTransaction
}

func (conn *RedisConnection) GetAddress() string {
	return conn.conn.RemoteAddr().String()
}

func (conn *RedisConnection) IsInPubSubMode() bool {
	return conn.inPubSubMode
}

func (conn *RedisConnection) IsMaster() bool {
	return conn.isMaster
}

// SendError sends an error response to the client
func (conn *RedisConnection) SendError(errMsg string) {
	conn.SendResponse(utils.ToError(errMsg))
}
