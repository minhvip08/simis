package handler

import (
	"github.com/minhvip08/simis/internal/connection"
	err "github.com/minhvip08/simis/internal/error"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type RPushHandler struct{}

type rPushParams struct {
	key    string
	values []string
}

func parserPushParams(args []string) (*rPushParams, error) {
	if len(args) < 2 {
		return nil, ErrInvalidArguments
	}

	key := args[0]
	values := args[1:]

	return &rPushParams{
		key:    key,
		values: values,
	}, nil
}

func executeRPush(params *rPushParams) *ExecutionResult {
	db := store.GetInstance()

	dp, _ := db.LoadOrStoreList(params.key)

	if dp == nil {
		result := NewExecutionResult()
		result.Error = err.ErrFailedToLoadOrStoreList
		return result
	}

	n := dp.PushBack(params.values...)
	result := NewExecutionResult()
	result.Response = utils.ToRespInt(n)
	result.ModifiedKeys = []string{params.key}
	return result
}

func (h *RPushHandler) Execute(cmd *connection.Command) *ExecutionResult {
	parmas, err := parserPushParams(cmd.Args)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}
	return executeRPush(parmas)
}
