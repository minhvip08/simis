package handler

import (
	"fmt"
	"strings"

	"github.com/minhvip08/simis/internal/config"
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type InfoHandler struct {
}

type infoParams struct {
	section string
}

func parseInfoParams(args []string) infoParams {
	if len(args) > 0 {
		return infoParams{section: strings.ToLower(args[0])}
	}
	return infoParams{section: ""}
}

func executeInfo(params infoParams) *ExecutionResult {
	cfg := config.GetInstance()
	result := NewExecutionResult()

	switch params.section {
	case "replication":
		result.Response = utils.ToBulkString(cfg.String())
	case "memory":
		result.Response = utils.ToBulkString(memoryInfoString(cfg))
	default:
		result.Response = utils.ToNullBulkString()
	}
	return result
}

func memoryInfoString(cfg *config.Config) string {
	db := store.GetInstance()
	return fmt.Sprintf(
		"# Memory\nused_memory_heap:%d\nmaxmemory:%d\nmaxmemory_policy:%s\nmem_evictions:%d",
		store.CurrentMemory(),
		cfg.MaxMemory,
		cfg.EvictionPolicy,
		db.TotalEvictions(),
	)
}

func (h *InfoHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params := parseInfoParams(cmd.Args)
	return executeInfo(params)
}
