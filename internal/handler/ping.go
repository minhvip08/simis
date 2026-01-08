package handler

import (
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/utils"
)

type PingHandler struct {
}

type pingParams struct{}

func parsePing(arg []string) (*pingParams, error) {
	return &pingParams{}, nil
}

func executePing(params *pingParams) *ExecutionResult {
	return &ExecutionResult{
		Response: utils.ToSimpleString("PONG"),
	}
}

func (h *PingHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parsePing(cmd.Args)
	if err != nil {
		return &ExecutionResult{
			Error: err,
		}
	}
	return executePing(params)
}
