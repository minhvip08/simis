package handler

import (
	"strings"

	"github.com/minhvip08/simis/internal/config"
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/utils"
)

type InfoHandler struct {
}

type infoParams struct {
	section string
}

func parseInfoParams(args []string) infoParams {
	if len(args) > 0 {
		return infoParams{section: args[0]}
	}
	return infoParams{section: ""}
}

func executeInfo(params infoParams) *ExecutionResult {
	cfg := config.GetInstance()
	result := NewExecutionResult()
	if strings.ToLower(params.section) == "replication" {
		result.Response = utils.ToBulkString(cfg.String())
	} else {
		result.Response = utils.ToNullBulkString()
	}
	return result
}

func (h *InfoHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params := parseInfoParams(cmd.Args)
	return executeInfo(params)
}
