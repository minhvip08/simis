package session

import "github.com/minhvip08/simis/internal/connection"

type SessionHandler interface {
	Execute(cmd *connection.Command, conn *connection.RedisConnection) error
}

var SessionHandlers = map[string]SessionHandler{
	"MULTI":       &MultiHandler{},
	"EXEC":        &ExecHandler{},
	"DISCARD":     &DiscardHandler{},
	"REPLCONF":    &ReplconfHandler{},
	"PSYNC":       &PSyncHandler{},
	"WAIT":        &WaitHandler{},
	"SUBSCRIBE":   &SubscribeHandler{},
	"UNSUBSCRIBE": &UnsubscribeHandler{},
}
