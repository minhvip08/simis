package handler

import "github.com/minhvip08/simis/app/connection"

// Handler interface defines the Execute method for command handlers
type Handler interface {
	Execute(cmd *connection.Command) *ExecutionResult
}

type ExecutionResult struct {
	Response string
	Error    error

	BlockingTimeout int
	WaitingKeys     []string // Blocking keys

	ModifiedKeys []string
}

var Handlers = map[string]Handler{
	"PING": &PingHandler{},
}
