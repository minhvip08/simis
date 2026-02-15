package handler

import "github.com/minhvip08/simis/internal/connection"

// Handler interface defines the Execute method for command handlers
type Handler interface {
	Execute(cmd *connection.Command) *ExecutionResult
}

type ExecutionResult struct {
	Response        string
	Error           error
	BlockingTimeout int      // if >= 0, indicates the command is blocking with the specified timeout in seconds
	WaitingKeys     []string // Blocking keys
	ModifiedKeys    []string // Keys that were modified during execution
}

// NewExecutionResult creates a new ExecutionResult with default values
// BlockingTimeout defaults to -1 (no blocking)
func NewExecutionResult() *ExecutionResult {
	return &ExecutionResult{
		BlockingTimeout: -1,
	}
}

func NewErrorResult(err error) *ExecutionResult {
	result := NewExecutionResult()
	result.Error = err
	return result
}

var Handlers = map[string]Handler{
	"PING":      &PingHandler{},
	"ECHO":      &EchoHandler{},
	"SET":       &SetHandler{},
	"GET":       &GetHandler{},
	"RPUSH":     &RPushHandler{},
	"LRANGE":    &LRangeHandler{},
	"LPUSH":     &LPushHandler{},
	"LLEN":      &LLENHandler{},
	"LPOP":      &LPopHandler{},
	"BLPOP":     &BLPopHandler{},
	"TYPE":      &TypeHandler{},
	"XADD":      &XAddHandler{},
	"XRANGE":    &XRangeHandler{},
	"XREAD":     &XReadHandler{},
	"INCR":      &IncrHandler{},
	"INFO":      &InfoHandler{},
	"CONFIG":    &ConfigHandler{},
	"KEYS":      &KeysHandler{},
	"BGSAVE":    &BGSaveHandler{},
	"PUBLISH":   &PublishHandler{},
	"ZADD":      &ZAddHandler{},
	"ZRANK":     &ZRankHandler{},
	"ZRANGE":    &ZRangeHandler{},
	"ZCARD":     &ZCardHandler{},
	"ZSCORE":    &ZScoreHandler{},
	"ZREM":      &ZRemHandler{},
	"GEOADD":    &GeoAddHandler{},
	"GEOPOS":    &GeoPosHandler{},
	"GEODIST":   &GeoDistHandler{},
	"GEOSEARCH": &GeoSearchHandler{},
}
