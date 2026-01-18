package handler

import (
	"strconv"

	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type LPopHandler struct{}

type lPopParams struct {
	key      string
	count    int
	hasCount bool
}

func parseLPop(args []string) (*lPopParams, error) {
	if len(args) < 1 {
		return nil, ErrInvalidArguments
	}

	params := &lPopParams{
		key: args[0],
	}

	if len(args) >= 2 {
		count, err := strconv.Atoi(args[1])
		if err != nil {
			return nil, ErrInvalidArguments
		}
		params.count = count
		params.hasCount = true
	}
	return params, nil
}

func executeLPop(params *lPopParams) *ExecutionResult {
	db := store.GetInstance()

	dp, exists := db.GetList(params.key)

	result := NewExecutionResult()
	if !exists || dp == nil {
		result.Response = utils.ToNullBulkString()
		return result
	}

	if params.hasCount {
		removedElements, nRemoved := dp.PopNFront(params.count)
		if nRemoved == 0 {
			result.Response = utils.ToNullBulkString()
		} else {
			result.Response = utils.ToRespArray(removedElements)
		}
	} else {
		value, ok := dp.PopFront()
		if !ok {
			result.Response = utils.ToNullBulkString()
		} else {
			result.Response = utils.ToBulkString(value)
		}
	}
	return result
}

func (h *LPopHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseLPop(cmd.Args)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}

	return executeLPop(params)
}
