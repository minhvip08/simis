package handler

import (
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type IncrHandler struct {
}

type incrParams struct {
	key string
}

func parseIncrParams(args []string) (*incrParams, error) {
	if len(args) != 1 {
		return nil, ErrInvalidArguments
	}
	return &incrParams{
		key: args[0],
	}, nil
}

func executeIncr(params *incrParams) *ExecutionResult {
	db := store.GetInstance()
	count, err := db.IncrementString(params.key)

	result := NewExecutionResult()
	if err != nil {
		result.Error = err
		return result
	}

	result.Response = utils.ToRespInt(count)
	return result
}

func (h *IncrHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseIncrParams(cmd.Args)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}
	return executeIncr(params)
}
