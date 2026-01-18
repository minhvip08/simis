package handler

import (
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/utils"
)

type EchoHandler struct{}

type echoParams struct {
	message string
}

func parseEcho(args []string) (*echoParams, error) {
	if len(args) == 0 {
		return nil, ErrInvalidArguments
	}
	return &echoParams{message: args[0]}, nil
}

func executeEcho(params *echoParams) *ExecutionResult {
	result := NewExecutionResult()
	result.Response = utils.ToBulkString(params.message)
	return result
}

func (h *EchoHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseEcho(cmd.Args)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}
	return executeEcho(params)
}
