package handler

import (
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type GetHandler struct{}

type getParams struct {
	key string
}

func parseGet(args []string) (*getParams, error) {
	if len(args) < 1 {
		return nil, ErrInvalidArguments
	}
	params := &getParams{
		key: args[0],
	}
	return params, nil
}

func executeGet(params *getParams) *ExecutionResult {
	db := store.GetInstance()
	kvObj, err := db.GetString(params.key)
	result := NewExecutionResult()
	if !err {
		result.Response = utils.ToNullBulkString()
	} else {
		result.Response = utils.ToBulkString(kvObj)
	}
	return result
}

func (h *GetHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseGet(cmd.Args)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}
	return executeGet(params)
}
