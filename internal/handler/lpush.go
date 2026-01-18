package handler

import (
	"slices"

	"github.com/minhvip08/simis/internal/connection"
	err "github.com/minhvip08/simis/internal/error"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type LPushHandler struct{}

type lPushParams struct {
	key   string
	value []string
}

func parseLPush(args []string) (*lPushParams, error) {
	if len(args) < 2 {
		return nil, ErrInvalidArguments
	}

	// Make a copy to avoid mutating input args
	values := make([]string, len(args)-1)
	copy(values, args[1:])
	slices.Reverse(values)

	return &lPushParams{
		key:   args[0],
		value: values,
	}, nil
}

func executeLPush(params *lPushParams) *ExecutionResult {
	db := store.GetInstance()

	dp, _ := db.LoadOrStoreList(params.key)

	if dp == nil {
		result := NewExecutionResult()
		result.Error = err.ErrFailedToLoadOrStoreList
		return result
	}

	newLength := dp.PushFront(params.value...)
	// for i := 0; i < len(params.value); i++ {
	// 	dp.PushFront(params.value[i])
	// }
	// newLength := dp.Length()

	result := NewExecutionResult()
	result.Response = utils.ToRespInt(newLength)
	result.ModifiedKeys = []string{params.key}
	return result
}

func (h *LPushHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseLPush(cmd.Args)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}

	return executeLPush(params)
}
