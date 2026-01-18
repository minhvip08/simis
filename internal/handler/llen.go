package handler

import (
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type LLENHandler struct{}

type lLenParams struct {
	key string
}

func parseLLen(args []string) (*lLenParams, error) {
	if len(args) != 1 {
		return nil, ErrInvalidArguments
	}

	return &lLenParams{
		key: args[0],
	}, nil
}

func executeLLen(params *lLenParams) *ExecutionResult {
	db := store.GetInstance()

	dp, exists := db.GetList(params.key)

	result := NewExecutionResult()
	if !exists || dp == nil {
		result.Response = utils.ToRespInt(0)
		return result
	}

	length := dp.Length()
	result.Response = utils.ToRespInt(length)
	return result
}

func (h *LLENHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseLLen(cmd.Args)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}

	return executeLLen(params)
}
